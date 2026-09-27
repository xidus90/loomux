// Package load reads a flow's folder and refuses it before anything runs. It
// decodes flow.toml into declarations, then walks six stages -- declarations,
// node kinds, texts, names, written fields, graph rules -- and reports every
// finding of the first stage that has one. It also finds flows by name in the
// project, under .loomux/flows, and in the catalog it is handed, and lays a
// project's overlay over a bundled flow.
//
// It is a package of its own and not part of internal/flow because it parses
// conditions and caps with internal/flow/expr, and expr imports internal/flow.
package load
