package mount

/*
#cgo pkg-config: glib-2.0
#include <glib.h>
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/linuxdeepin/go-gir/gio-2.0"
	"github.com/linuxdeepin/go-gir/gobject-2.0"
	log "github.com/ubuntu/adsys/internal/grpc/logstreamer"
)

// mountEntry represents a parsed entry to be mounted.
type mountEntry struct {
	path    string
	krbAuth bool
}

// msg struct is the message structure that will be used to communicate in the mountsChan channel.
type msg struct {
	path string
	err  error
}

// mountsChan is the channel through which the async mount operations will communicate with the main
// routine.
var mountsChan chan msg

// RunMountForCurrentUser reads the specified file and tries to mount the parsed entries for the
// current user.
func RunMountForCurrentUser(ctx context.Context, filepath string) error {
	log.Debugf(ctx, "Reading mount entries from %q", filepath)
	entries, err := parseEntries(filepath)
	if err != nil {
		return err
	}

	mountsChan = make(chan msg, len(entries))

	var wg sync.WaitGroup
	for _, entry := range entries {
		entry := entry
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Debugf(ctx, "Creating mount operation for %q", entry.path)
			setupMountOperation(entry)
		}()
	}
	wg.Wait()

	mainLoop := C.g_main_loop_new(C.g_main_context_default(), C.FALSE)

	// watches the mountsChan channel for the results of the mount operations.
	go func(c chan msg, count int) {
		for i := 0; i < count; {
			m := <-c
			logMsg := fmt.Sprintf("Successfully mounted %q", m.path)
			if m.err != nil {
				logMsg = fmt.Sprintf("Failed to mount %q: %v", m.path, m.err)
				err = errors.Join(err, m.err)
			}
			log.Debugf(ctx, logMsg)
			i++
		}
		C.g_main_loop_quit(mainLoop)
	}(mountsChan, len(entries))

	C.g_main_loop_run(mainLoop)
	return err
}

// parseEntries reads the specified file and parses the listed mount locations from it.
func parseEntries(filepath string) ([]mountEntry, error) {
	var entries []mountEntry

	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		line, krb := strings.CutPrefix(line, "[krb5]")
		entries = append(entries, mountEntry{path: line, krbAuth: krb})
	}

	return entries, nil
}

// setupMountOperation creates and starts a gio mount operation for the specified location.
func setupMountOperation(entry mountEntry) {
	f := gio.FileNewForUri(entry.path)

	op := gio.NewMountOperation()
	op.SetAnonymous(true)
	if entry.krbAuth {
		op.SetAnonymous(false)
	}
	op.Connect("ask_password", askPassword)

	f.MountEnclosingVolume(gio.MountMountFlagsNone, op, gio.CancellableGetCurrent(), mountDone)
}

// askPassword is the callback that is connected to the ask_password signal of the specified mount
// operation.
func askPassword(op *gio.MountOperation, _, _, _ string, flags gio.AskPasswordFlags) {
	if op.GetAnonymous() && (gio.AskPasswordFlagsNeedPassword&flags == 1) {
		op.Reply(gio.MountOperationResultHandled)
		return
	}

	if krb := os.Getenv("KRB5CCNAME"); krb != "" {
		op.Reply(gio.MountOperationResultHandled)
		return
	}

	op.Reply(gio.MountOperationResultAborted)
}

// mountDone is the callback that is invoked once the mount operation is done.
func mountDone(src *gobject.Object, res *gio.AsyncResult) {
	f := gio.ToFile(src)
	_, err := f.MountEnclosingVolumeFinish(res)
	mountsChan <- msg{f.GetUri(), err}
}
