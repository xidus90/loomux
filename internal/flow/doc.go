// Package flow holds what a flow is made of: typed state, the loaded graph,
// and the contracts every block, condition and registry is built against.
//
// The contracts are fixed before any package that uses them is written, so the
// expression language, the loader, the blocks and the runner can be built side
// by side. The loader that fills a Graph from a flow's folder lives in
// internal/flow/load, because it parses conditions with internal/flow/expr,
// which imports this package.
package flow
