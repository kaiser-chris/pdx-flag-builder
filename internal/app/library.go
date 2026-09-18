package app

import (
	"cmp"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/kaiser-chris/pdx-flag-builder-go/internal/config"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/database"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
)

// library is everything read from the configured folders, and the background
// read that produces it.
//
// A full game folder is thousands of files, which is too long to stall a frame
// for, so the reading happens on its own goroutine and the result arrives
// through a channel the frame loop checks once a frame. Nothing is shared with
// that goroutine while it runs, so none of this needs a lock.
type library struct {
	set      database.Set
	problems []database.Problem

	// Flattened for the interface, which wants one sorted list rather than one
	// per folder.
	flags    []pdx.Flag
	textures []database.Texture
	palette  pdx.Palette

	// paletteNames are the named colours in alphabetical order, for the colour
	// picker.
	paletteNames []string

	loading bool
	result  chan loadResult
	took    time.Duration

	// again records that the folders changed while a read was still running.
	again bool

	// version counts finished reads, so that anything derived from the data can
	// tell when it has gone stale.
	version int
}

type loadResult struct {
	set      database.Set
	problems []database.Problem
	took     time.Duration
}

// reload starts reading the configured folders. Asking again while a read is
// running queues another one rather than racing it.
func (l *library) reload(folders []config.Database) {
	if l.loading {
		l.again = true

		return
	}

	targets := make([]database.Folder, 0, len(folders))
	for _, folder := range folders {
		if folder.Path == "" {
			continue
		}

		targets = append(targets, database.Folder{Name: folder.Name, Path: folder.Path})
	}

	l.loading = true
	l.again = false
	l.result = make(chan loadResult, 1)

	go func(result chan<- loadResult) {
		started := time.Now()
		set, problems := database.LoadAll(targets)

		result <- loadResult{set: set, problems: problems, took: time.Since(started)}
	}(l.result)
}

// poll takes the result of a finished read. It reports whether one arrived.
func (l *library) poll(folders []config.Database) bool {
	if !l.loading {
		return false
	}

	select {
	case result := <-l.result:
		l.set = result.set
		l.problems = result.problems
		l.took = result.took
		l.flags = result.set.Flags()
		l.textures = result.set.Textures()
		l.palette = result.set.Palette()
		l.paletteNames = slices.Sorted(maps.Keys(l.palette))
		l.loading = false
		l.result = nil
		l.version++

		if l.again {
			l.reload(folders)
		}

		return true

	default:
		return false
	}
}

// filteredRows keeps the rows a search box matches.
//
// The lists are long enough that filtering them every frame would be wasteful,
// so the result is kept until either the search or the data behind it changes.
type filteredRows struct {
	query   string
	version int
	valid   bool
	rows    []int

	// order is what the rows are sorted by, once sorted is true.
	order  tableOrder
	sorted bool
}

// get returns the indexes matching the query. matches is asked only about rows
// that have to be reconsidered, and is given the query already lowercased and
// trimmed. It is asked even for an empty query, which a list narrowed down by
// something besides the search still has to apply.
func (f *filteredRows) get(query string, version, count int, matches func(index int, query string) bool) []int {
	if f.valid && f.query == query && f.version == version {
		return f.rows
	}

	f.query = query
	f.version = version
	f.valid = true
	f.rows = f.rows[:0]
	f.sorted = false

	lowered := strings.ToLower(strings.TrimSpace(query))

	for index := range count {
		if matches(index, lowered) {
			f.rows = append(f.rows, index)
		}
	}

	return f.rows
}

// invalidate forces the next call to get to filter again.
func (f *filteredRows) invalidate() {
	f.valid = false
}

func containsFold(haystack, lowercaseNeedle string) bool {
	return strings.Contains(strings.ToLower(haystack), lowercaseNeedle)
}

// inOrder sorts the rows found by the last get. compare compares two rows by
// a column. Rows that compare equal keep the order they were found in, which
// for every list here is by name. Like the filtering, the sorting is kept
// until the order or the rows change.
func (f *filteredRows) inOrder(order tableOrder, compare func(first, second, column int) int) []int {
	if f.sorted && f.order == order {
		return f.rows
	}

	f.order = order
	f.sorted = true

	if order.column < 0 {
		slices.Sort(f.rows)

		return f.rows
	}

	slices.SortFunc(f.rows, func(first, second int) int {
		result := compare(first, second, order.column)
		if order.descending {
			result = -result
		}

		if result == 0 {
			return cmp.Compare(first, second)
		}

		return result
	})

	return f.rows
}

// compareFold compares two strings the way a user reads them, ignoring case.
func compareFold(first, second string) int {
	return strings.Compare(strings.ToLower(first), strings.ToLower(second))
}
