/*
Package plugin defines a set of interfaces that plugins must implement to enable wash
functonality.

All resources must implement the Entry interface. To do so they should include the EntryBase
type, and initialize it via NewEntry. For example
	type myResource struct {
		plugin.EntryBase
	}
	...
	rsc := myResource{plugin.NewEntry("a resource")}
EntryBase gives the resource a name - which is how it will be displayed in the filesystem
or referenced via the API - and tools for controlling how its data is cached.

Implementing the Parent interface displays that resource as a directory on the filesystem.
Anything that does not implement Parent will be displayed as a file.

The Readable interface gives a file its contents when read via the filesystem.

All of the above, as well as other types - Execable, Stream - provide additional functionality
via the HTTP API.
*/
package plugin

// This file should be reserved for types that plugin authors need to understand.

import (
	"context"
	"io"
	"time"
)

// Entry is the interface for things that are representable by Wash's filesystem. This includes
// plugin roots; resources like containers and volumes; placeholders (e.g. the containers directory
// in the Docker plugin); read-only files like the metadata.json files for containers and EC2
// instances; and more. It is a sealed interface, meaning you must use plugin.NewEntry when
// creating your plugin objects.
//
// Metadata returns a complete description of the entry. See the EntryBase documentation for more
// details on when to override it.
type Entry interface {
	Metadata(ctx context.Context) (JSONObject, error)
	name() string
	attributes() EntryAttributes
	slashReplacer() rune
	id() string
	setID(id string)
	getTTLOf(op defaultOpCode) time.Duration
}

// Parent is an entry with children. It will be represented as a directory in the Wash
// filesystem.
type Parent interface {
	Entry
	List(context.Context) ([]Entry, error)
}

// Root represents the plugin root
type Root interface {
	Parent
	Init() error
}

// ExecOptions is a struct we can add new features to that must be serializable to JSON.
// Examples of potential features: user, privileged, map of environment variables, timeout.
type ExecOptions struct {
	// Stdin can be used to pass a stream of input to write to stdin when executing the command.
	// It is not included in ExecOption's JSON serialization.
	Stdin io.Reader `json:"-"`

	// Tty instructs the executor to allocate a TTY (pseudo-terminal), which lets Wash communicate
	// with the running process via its Stdin. The TTY is used to send a process termination signal
	// (Ctrl+C) via Stdin when the passed-in Exec context is cancelled.
	//
	// NOTE TO PLUGIN AUTHORS: The Tty option is only relevant for executors that do not have an API
	// endpoint to stop a running command (e.g. Docker, Kubernetes). If your executor does have an
	// API endpoint to stop a running command, then ignore the Tty option. Note that the reason we
	// make Tty an option instead of having the relevant executors always attach a TTY is because
	// attaching a TTY can change the behavior of the command that's being executed.
	//
	// NOTE TO CALLERS: The Tty option is useful for executing your own stream-like commands (e.g.
	// tail -f), because it ensures that there are no orphaned processes after the request is
	// cancelled/finished.
	Tty bool `json:"tty"`

	// Elevate execution to run as a privileged user if not already running as a privileged user.
	Elevate bool `json:"elevate"`
}

// ExecPacketType identifies the packet type.
type ExecPacketType = string

// Enumerates packet types.
const (
	Stdout ExecPacketType = "stdout"
	Stderr ExecPacketType = "stderr"
)

// ExecOutputChunk is a struct containing a chunk of the Exec'ed cmd's output.
type ExecOutputChunk struct {
	StreamID  ExecPacketType
	Timestamp time.Time
	Data      string
	Err       error
}

// ExecCommand represents a command that was invoked by a call to Exec.
// It is a sealed interface, meaning you must use plugin.NewExecCommand
// to create instances of these objects.
//
// OutputCh returns a channel containing timestamped chunks of the command's
// stdout/stderr.
//
// ExitCode returns the command's exit code. It will block until the command's
// exit code is set, or until the execution context is cancelled. ExitCode will
// return an error if it fails to fetch the command's exit code.
type ExecCommand interface {
	OutputCh() <-chan ExecOutputChunk
	ExitCode() (int, error)
	sealed()
}

// Execable is an entry that can have a command run on it.
type Execable interface {
	Entry
	Exec(ctx context.Context, cmd string, args []string, opts ExecOptions) (ExecCommand, error)
}

// Streamable is an entry that returns a stream of updates.
type Streamable interface {
	Entry
	Stream(context.Context) (io.ReadCloser, error)
}

// SizedReader returns a ReaderAt that can report its Size.
type SizedReader interface {
	io.ReaderAt
	Size() int64
}

// Readable is an entry that has a fixed amount of content we can read.
type Readable interface {
	Entry
	Open(context.Context) (SizedReader, error)
}
