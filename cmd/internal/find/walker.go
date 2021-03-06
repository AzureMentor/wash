package find

import (
	"fmt"

	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
)

type walker struct {
	p    types.EntryPredicate
	opts types.Options
	conn client.Client
}

func newWalker(r parser.Result, conn client.Client) *walker {
	return &walker{
		p:    r.Predicate,
		opts: r.Options,
		conn: conn,
	}
}

// Walk returns true if the walk is successful (i.e. does not
// have any errors), false otherwise.
func (w *walker) Walk(path string) bool {
	e, err := info(w.conn, path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return false
	}
	return w.walk(e, 0)
}

func (w *walker) walk(e types.Entry, depth uint) bool {
	// If the Depth option is set, then we visit e after visiting its children.
	// Otherwise, we visit e first.
	//
	// TODO: Write unit tests for the walker by mocking out the client.
	successful := true
	check := func(result bool) {
		// Use "&&" to short-circuit if successful is false
		successful = successful && result
	}
	if !w.opts.Depth {
		check(w.visit(e, depth))
	}
	childDepth := depth + 1
	if int(childDepth) <= w.opts.Maxdepth && e.Supports(plugin.ListAction()) {
		children, err := list(w.conn, e)
		if err != nil {
			cmdutil.ErrPrintf("could not get children of %v: %v\n", e.NormalizedPath, err)
			successful = false
		} else {
			for _, child := range children {
				check(w.walk(child, childDepth))
			}
		}
	}
	if w.opts.Depth {
		check(w.visit(e, depth))
	}
	return successful
}

func (w *walker) visit(e types.Entry, depth uint) bool {
	if depth < w.opts.Mindepth {
		return true
	}
	if primary.IsSet(primary.Meta) && w.opts.IsSet(types.FullmetaFlag) {
		// Fetch the entry's full metadata
		meta, err := w.conn.Metadata(e.Path)
		if err != nil {
			cmdutil.ErrPrintf("could not get full metadata of %v: %v\n", e.NormalizedPath, err)
			return false
		}
		e.Metadata = meta
	}
	if w.p(e) {
		fmt.Printf("%v\n", e.NormalizedPath)
	}
	return true
}
