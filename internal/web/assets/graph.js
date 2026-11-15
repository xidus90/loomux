// The graph page's island: fetches the view, hands it to force-graph and
// switches visibility. Folding, resolving and searching are the server's.
(function () {
  'use strict';

  const SETTLE_TICKS = 200;
  const DOUBLE_CLICK_MS = 300;
  const HIGHLIGHT = '#e8a400';

  const $ = (id) => document.getElementById(id);
  const canvas = $('graph-canvas');
  const project = encodeURIComponent($('graph').dataset.project);

  let view = { nodes: [], edges: [], open: [] };
  let byId = new Map();
  let links = [];
  let off = new Set();
  let selected = null; // the node highlighted and shown in the info box
  let centre = null; // label of the search hit whose node is to be centred
  let fit = false; // the next settled layout is fitted to the canvas
  let lastClick = { id: null, at: 0 };
  let loads = 0;
  let searches = 0;
  const infoHint = $('graph-info').firstElementChild;

  // el makes an element whose text is text, never markup.
  function el(tag, text, parent, cls) {
    const e = document.createElement(tag);
    if (text) e.textContent = text;
    if (cls) e.className = cls;
    if (parent) parent.append(e);
    return e;
  }

  // The tooltip library takes node labels as HTML, so they are escaped.
  const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

  async function getJSON(url) {
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(url + ': ' + resp.status);
    return resp.json();
  }

  // State lives in the fragment: #open=a,b&off=calls,docs. Without open (null)
  // the server opens directories itself, while an empty open shows the top
  // level only. Without off, other documents start hidden.
  function readHash() {
    const params = new Map(location.hash.slice(1).split('&').filter(Boolean).map((kv) => kv.split(/=(.*)/s)));
    const list = (k) => (params.has(k) ? (params.get(k) || '').split(',').filter(Boolean).map(decodeURIComponent) : null);
    return { open: list('open'), off: list('off') || ['docs'] };
  }

  const enc = (list) => list.map(encodeURIComponent).join(',');
  const hashOf = (open) => '#' + (open === null ? '' : 'open=' + enc(open) + '&') + 'off=' + enc([...off]);

  // navigate changes what is open: a new history entry, and a reload, which the
  // hash change brings unless the address stays the same.
  function navigate(open) {
    const before = location.href;
    location.hash = hashOf(open);
    if (location.href === before) load();
  }

  function showProblems(problems) {
    const box = $('graph-problems');
    box.replaceChildren();
    problems.forEach((p) => el('p', p, box, 'error'));
  }

  function color(group) {
    let h = 0;
    for (const c of group) h = (h * 31 + c.charCodeAt(0)) % 360;
    return 'hsl(' + h + ',65%,55%)';
  }

  const hidden = (n) => off.has('group:' + n.group) || (n.kind === 'doc' && off.has(n.wiki ? 'wiki' : 'docs'));
  const endId = (end) => (typeof end === 'object' ? end.id : end);

  // reach is the sum of an edge's counts that no filter switches off; a link or
  // ref edge follows its documents instead.
  function reach(l) {
    if (hidden(byId.get(endId(l.source))) || hidden(byId.get(endId(l.target)))) return 0;
    let sum = 0;
    for (const [key, n] of Object.entries(l.counts)) {
      const [relation, confidence] = key.split('/');
      if (l.kind !== 'code' || !(off.has(relation) || off.has(confidence))) sum += n;
    }
    return sum;
  }

  // draw hands the library only what no filter hides, so hidden nodes neither
  // pull at the layout nor count when it is fitted. It restarts the layout.
  function draw() {
    links.forEach((l) => { l.reach = reach(l); });
    graph.graphData({ nodes: view.nodes.filter((n) => !hidden(n)), links: links.filter((l) => l.reach > 0) });
  }

  // refresh hands the library fresh accessors, which is what makes it redraw.
  function refresh() {
    graph
      .linkWidth((l) => 1 + Math.log2(Math.max(l.reach, 1)))
      .nodeColor((n) => (n.id === selected ? HIGHLIGHT : color(n.group)));
  }

  function checkbox(parent, key, text, swatch) {
    const label = el('label', '', parent);
    const box = document.createElement('input');
    box.type = 'checkbox';
    box.checked = !off.has(key);
    box.addEventListener('change', () => {
      if (box.checked) off.delete(key);
      else off.add(key);
      history.replaceState(null, '', hashOf(readHash().open));
      draw();
      refresh();
    });
    label.append(box);
    if (swatch) el('span', '', label, 'swatch').style.background = swatch;
    label.append(' ' + text);
  }

  function drawControls() {
    const relations = new Set();
    const confidences = new Set();
    for (const e of view.edges) {
      if (e.kind !== 'code') continue;
      for (const key of Object.keys(e.counts)) {
        const [relation, confidence] = key.split('/');
        relations.add(relation);
        if (confidence) confidences.add(confidence);
      }
    }
    const filters = $('graph-filters');
    filters.replaceChildren(el('legend', 'Filters'));
    el('h3', 'Relations', filters);
    [...relations].sort().forEach((k) => checkbox(filters, k, k));
    el('h3', 'Confidence', filters);
    [...confidences].sort().forEach((k) => checkbox(filters, k, k));
    el('h3', 'Documents', filters);
    checkbox(filters, 'wiki', 'Wiki');
    checkbox(filters, 'docs', 'Other documents');

    const legend = $('graph-legend');
    legend.replaceChildren(el('legend', 'Legend'));
    [...new Set(view.nodes.map((n) => n.group))].sort().forEach((g) => checkbox(legend, 'group:' + g, g || '(none)', color(g)));
  }

  // The path bar lists what is open; a click closes that entry and the ones below.
  function drawPath() {
    const bar = $('graph-path');
    bar.replaceChildren();
    if (view.open.length) bar.append('Open: ');
    for (const p of view.open) {
      const b = el('button', p + ' ×', bar);
      b.type = 'button';
      b.title = 'Close ' + p;
      b.addEventListener('click', () => navigate(view.open.filter((o) => o !== p && !o.startsWith(p + '/'))));
    }
  }

  function showInfo(n) {
    const box = $('graph-info');
    box.replaceChildren();
    el('h3', n.label, box);
    const facts = el('ul', '', box);
    el('li', 'Kind: ' + n.kind, facts);
    el('li', 'Path: ' + n.path, facts);
    el('li', 'Members: ' + n.members, facts);
    if (n.missing && n.missing.length) {
      el('h3', 'Not in the code graph', box);
      const missing = el('ul', '', box);
      n.missing.forEach((m) => el('li', m, missing));
    }
    const near = el('ul', '', box);
    for (const e of view.edges) {
      if (e.source !== n.id && e.target !== n.id) continue;
      const out = e.source === n.id;
      const other = byId.get(out ? e.target : e.source);
      const counts = Object.entries(e.counts).map(([k, v]) => k + ' ' + v).join(', ');
      el('li', (out ? '→ ' : '← ') + (other ? other.label : '?') + ' (' + counts + ')', near);
    }
    if (near.children.length) box.insertBefore(el('h3', 'Neighbours'), near);
    else near.remove();
  }

  function onClick(n) {
    const now = Date.now();
    const double = lastClick.id === n.id && now - lastClick.at < DOUBLE_CLICK_MS;
    lastClick = { id: double ? null : n.id, at: now };
    selected = n.id;
    showInfo(n);
    refresh();
    if (double && (n.kind === 'dir' || n.kind === 'file')) navigate(view.open.concat(n.path));
  }

  const graph = ForceGraph()($('graph-canvas'))
    .width(canvas.clientWidth)
    .height(canvas.clientHeight)
    .nodeId('id')
    .nodeVal('size')
    .nodeLabel((n) => esc(n.label + ' (' + n.kind + ')'))
    .linkColor(() => 'rgba(128,128,128,0.6)')
    .cooldownTicks(SETTLE_TICKS)
    .onNodeClick(onClick)
    .onEngineStop(() => {
      const n = centre && byId.get(selected);
      if (n) {
        graph.centerAt(n.x, n.y, 400);
        graph.zoom(4, 400);
      } else if (fit) {
        graph.zoomToFit(400, 40, (m) => !hidden(m));
      }
      centre = null;
      fit = false;
    });

  async function load() {
    const ticket = ++loads;
    try {
      const state = readHash();
      off = new Set(state.off);
      // An empty list goes out as a bare `open=`, so the server tells it from none.
      const query = state.open === null ? '' : (state.open.length ? state.open : ['']).map((p) => 'open=' + encodeURIComponent(p)).join('&');
      const reply = await getJSON('/graph/' + project + '/data?' + query);
      if (ticket !== loads) return;
      view = reply.view;
      byId = new Map(view.nodes.map((n) => [n.id, n]));
      links = view.edges.map((e) => Object.assign({}, e));
      const problems = reply.problems.slice();
      if (centre) {
        const n = byId.get(selected);
        if (n && hidden(n) && n.kind === 'doc' && !n.wiki && off.has('docs')) {
          off.delete('docs');
          history.replaceState(null, '', hashOf(state.open));
        }
        if (!n || hidden(n)) {
          problems.push(centre + (n ? ' is hidden by a filter' : ' is not in the view: too many nodes are open'));
          centre = null;
        }
      }
      if (!byId.has(selected)) {
        selected = null;
        $('graph-info').replaceChildren(infoHint);
      }
      showProblems(problems);
      drawControls();
      drawPath();
      fit = true;
      draw();
      refresh();
      if (selected) showInfo(byId.get(selected));
    } catch (err) {
      showProblems([String(err)]);
    }
  }

  const form = $('graph-search');
  let typing = 0;

  async function search() {
    const ticket = ++searches;
    const hits = $('graph-hits');
    hits.replaceChildren();
    const q = form.q.value.trim();
    if (!q) return;
    try {
      const found = await getJSON('/graph/' + project + '/search?q=' + encodeURIComponent(q));
      if (ticket !== searches) return;
      if (!found.length) el('li', 'No hits', hits, 'note');
      for (const h of found) {
        const b = el('button', h.label + ' (' + h.kind + ')', el('li', '', hits));
        b.type = 'button';
        b.addEventListener('click', () => {
          selected = h.id;
          centre = h.label;
          hits.replaceChildren();
          // Every view holds every document, so a document leaves the view as it is.
          navigate(h.kind === 'doc' ? view.open : h.open);
        });
      }
    } catch (err) {
      if (ticket === searches) showProblems([String(err)]);
    }
  }

  form.addEventListener('submit', (e) => { e.preventDefault(); search(); });
  form.addEventListener('input', () => { clearTimeout(typing); typing = setTimeout(search, 250); });
  window.addEventListener('hashchange', load);
  window.addEventListener('resize', () => graph.width(canvas.clientWidth).height(canvas.clientHeight));
  load();
})();
