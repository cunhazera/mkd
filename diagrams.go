package main

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// ANSI colours used exclusively for diagram rendering.
const (
	diagTitleANSI = "\x1b[38;5;99m\x1b[1m" // bold purple — diagram type label
	diagBoxANSI   = "\x1b[38;5;75m"          // blue         — boxes / borders
	diagLineANSI  = "\x1b[38;5;241m"         // gray         — lifelines / shafts
)

// renderMermaidBlock is called from extractCodeBlocks when lang == "mermaid".
func renderMermaidBlock(code string, width int) string {
	lines := strings.Split(strings.TrimSpace(code), "\n")
	if len(lines) == 0 {
		return "\n"
	}
	first := strings.ToLower(strings.TrimSpace(lines[0]))
	switch {
	case first == "sequencediagram":
		if s := renderSequenceDiagram(lines[1:], width); s != "" {
			return s
		}
	case strings.HasPrefix(first, "graph ") || strings.HasPrefix(first, "flowchart "):
		if s := renderFlowchart(lines, width); s != "" {
			return s
		}
	}
	return renderMermaidFallback(code)
}

func renderMermaidFallback(code string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(diagTitleANSI + "  ⬡ Mermaid Diagram" + ansiReset + "\n")
	for _, line := range strings.Split(strings.TrimRight(code, "\n"), "\n") {
		sb.WriteString(padWithBackground("  "+line, termWidth()) + "\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

// ── rune-grid helpers ────────────────────────────────────────────────────────

func mkRow(w int) []rune {
	row := make([]rune, w)
	for i := range row {
		row[i] = ' '
	}
	return row
}

func setR(row []rune, x int, r rune) {
	if x >= 0 && x < len(row) {
		row[x] = r
	}
}

// setS writes the runes of s into row starting at x, returns next x.
func setS(row []rune, x int, s string) int {
	for _, r := range s {
		if x >= 0 && x < len(row) {
			row[x] = r
		}
		x++
	}
	return x
}

func emitRow(sb *strings.Builder, row []rune, color string) {
	sb.WriteString(color + string(row) + ansiReset + "\n")
}

// ── Sequence Diagram ─────────────────────────────────────────────────────────

var seqParticipantRe = regexp.MustCompile(`(?i)^(?:participant|actor)\s+(\S+)(?:\s+as\s+(.+))?$`)

type seqMsg struct {
	from, to string
	text     string
	dashed   bool
	self     bool
}

// parseSeqLine parses "A->>B: text" style lines.
func parseSeqLine(line string) (from, to, text string, dashed, ok bool) {
	ci := strings.Index(line, ":")
	if ci < 0 {
		return
	}
	text = strings.TrimSpace(line[ci+1:])
	lhs := strings.TrimSpace(line[:ci])

	di := strings.IndexByte(lhs, '-')
	if di < 0 {
		return
	}
	from = strings.TrimSpace(lhs[:di])
	if from == "" {
		return
	}

	// Advance past the arrow characters (-, >, x, ), +, .)
	i := di
	for i < len(lhs) {
		b := lhs[i]
		if b == '-' || b == '>' || b == 'x' || b == ')' || b == '+' || b == '.' {
			i++
		} else {
			break
		}
	}
	arrow := lhs[di:i]
	to = strings.TrimLeft(strings.TrimSpace(lhs[i:]), "+-")

	if from == "" || to == "" || len(arrow) < 2 {
		return
	}
	dashed = strings.Contains(arrow, "--")
	ok = true
	return
}

func renderSequenceDiagram(lines []string, width int) string {
	order := []string{}
	labelOf := map[string]string{}
	var msgs []seqMsg

	addActor := func(id string) {
		if _, ok := labelOf[id]; !ok {
			labelOf[id] = id
			order = append(order, id)
		}
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		if m := seqParticipantRe.FindStringSubmatch(line); m != nil {
			id := m[1]
			lbl := strings.TrimSpace(m[2])
			if lbl == "" {
				lbl = id
			}
			addActor(id)
			labelOf[id] = lbl
			continue
		}
		// Skip control-flow and decoration keywords.
		kw := strings.ToLower(strings.Fields(line)[0])
		switch kw {
		case "loop", "alt", "else", "opt", "par", "and", "critical",
			"break", "rect", "end", "note", "activate", "deactivate":
			continue
		}

		from, to, text, dashed, ok := parseSeqLine(line)
		if !ok {
			continue
		}
		addActor(from)
		addActor(to)
		msgs = append(msgs, seqMsg{
			from: from, to: to, text: text,
			dashed: dashed, self: from == to,
		})
	}

	if len(order) == 0 || len(msgs) == 0 {
		return ""
	}

	idx := map[string]int{}
	for i, id := range order {
		idx[id] = i
	}
	labels := make([]string, len(order))
	for i, id := range order {
		labels[i] = labelOf[id]
	}

	// ── Layout constants ──────────────────────────────────────────────────────
	const indent = 2
	const minGap = 12

	maxLabel := 0
	for _, l := range labels {
		if n := utf8.RuneCountInString(l); n > maxLabel {
			maxLabel = n
		}
	}
	maxMsg := 0
	for _, m := range msgs {
		if !m.self {
			if n := utf8.RuneCountInString(m.text); n > maxMsg {
				maxMsg = n
			}
		}
	}

	boxW := maxLabel + 4
	if boxW < 8 {
		boxW = 8
	}
	if boxW%2 != 0 {
		boxW++
	}

	gap := maxMsg + 6
	if gap < minGap {
		gap = minGap
	}
	n := len(order)
	if n > 1 {
		maxGap := (width - indent - boxW) / (n - 1)
		if maxGap >= minGap && gap > maxGap {
			gap = maxGap
		}
	}

	centers := make([]int, n)
	for i := range order {
		centers[i] = indent + boxW/2 + i*gap
	}
	totalW := centers[n-1] + boxW/2 + 2

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(diagTitleANSI + "  ⬡ Sequence Diagram" + ansiReset + "\n\n")

	// ── Actor header boxes ────────────────────────────────────────────────────

	// Top border: ┌──────┐
	row := mkRow(totalW)
	for _, c := range centers {
		half := boxW / 2
		s := c - half
		setR(row, s, '┌')
		for x := s + 1; x < s+boxW-1; x++ {
			setR(row, x, '─')
		}
		setR(row, s+boxW-1, '┐')
	}
	emitRow(&sb, row, diagBoxANSI)

	// Label row: │ Name │
	row = mkRow(totalW)
	for i, c := range centers {
		half := boxW / 2
		s := c - half
		setR(row, s, '│')
		setR(row, s+boxW-1, '│')
		lbl := labels[i]
		lw := utf8.RuneCountInString(lbl)
		inner := boxW - 2
		lpad := (inner - lw) / 2
		setS(row, s+1+lpad, lbl)
	}
	emitRow(&sb, row, diagBoxANSI)

	// Bottom border with downward stem: └──┬──┘
	row = mkRow(totalW)
	for _, c := range centers {
		half := boxW / 2
		s := c - half
		setR(row, s, '└')
		for x := s + 1; x < c; x++ {
			setR(row, x, '─')
		}
		setR(row, c, '┬')
		for x := c + 1; x < s+boxW-1; x++ {
			setR(row, x, '─')
		}
		setR(row, s+boxW-1, '┘')
	}
	emitRow(&sb, row, diagBoxANSI)

	// ── Messages ──────────────────────────────────────────────────────────────

	emitLifelines := func() {
		row = mkRow(totalW)
		for _, c := range centers {
			setR(row, c, '│')
		}
		emitRow(&sb, row, diagLineANSI)
	}

	for _, msg := range msgs {
		fi := idx[msg.from]
		ti := idx[msg.to]

		emitLifelines()

		if msg.self {
			// Self-arrows are complex to draw; show a note line instead.
			row = mkRow(totalW)
			for _, c := range centers {
				setR(row, c, '│')
			}
			c := centers[fi]
			setR(row, c, '↺')
			note := " " + msg.text
			setS(row, c+1, note)
			emitRow(&sb, row, diagLineANSI)
			continue
		}

		row = mkRow(totalW)
		for _, c := range centers {
			setR(row, c, '│')
		}

		rightward := ti > fi
		var x1, x2 int
		if rightward {
			x1, x2 = centers[fi], centers[ti]
		} else {
			x1, x2 = centers[ti], centers[fi]
		}

		shaft := '─'
		if msg.dashed {
			shaft = '╌'
		}

		// Draw shaft across entire span (overwrites lifelines of in-between actors).
		for x := x1 + 1; x < x2; x++ {
			setR(row, x, shaft)
		}

		// Arrowhead and source anchor.
		if rightward {
			setR(row, x1, '│') // source lifeline
			setR(row, x2, '▶') // tip at destination
		} else {
			setR(row, x1, '◀') // tip at destination (left side)
			setR(row, x2, '│') // source lifeline
		}

		// Embed text centred in the shaft.
		textRunes := []rune(msg.text)
		shaftLen := x2 - x1 - 1 // available interior characters
		if len(textRunes) > 0 && shaftLen > 2 {
			maxTxt := shaftLen - 2
			if len(textRunes) > maxTxt {
				textRunes = append(textRunes[:maxTxt-1], '…')
			}
			tstart := x1 + 1 + (shaftLen-len(textRunes))/2
			for i, r := range textRunes {
				pos := tstart + i
				if pos >= x2 {
					break
				}
				setR(row, pos, r)
			}
		}

		emitRow(&sb, row, diagLineANSI)
	}

	emitLifelines()
	sb.WriteString("\n")
	return sb.String()
}

// ── Flowchart ────────────────────────────────────────────────────────────────

var (
	fcSubgraphRe = regexp.MustCompile(`(?i)^subgraph\s+(\S+?)(?:\s+\[([^\]]+)\])?$`)
	fcBrRe       = regexp.MustCompile(`(?i)<br\s*/?>`)
	fcHTMLTagRe  = regexp.MustCompile(`<[^>]+>`)
	// Matches: ID[label], ID{label}, ID((label)), ID(label), or bare ID.
	fcNodeRe = regexp.MustCompile(`^(\w+)(\[([^\]]*)\]|\{([^}]*)\}|\(\(([^)]*)\)\)|\(([^)]*)\))?$`)
)

type fcNode struct{ id, label, kind string } // kind: rect | diamond | round | db
type fcEdge struct{ from, to, label string }

type sgBlock struct {
	id, label string
	nodeOrder []string
	nodes     map[string]fcNode
	edges     []fcEdge
}

func newSGBlock(id, label string) *sgBlock {
	if label == "" {
		label = id
	}
	return &sgBlock{id: id, label: label, nodes: map[string]fcNode{}}
}

// cleanFCLabel strips HTML tags and <br/> from node label text.
func cleanFCLabel(s string) string {
	s = fcBrRe.ReplaceAllString(s, " / ")
	s = fcHTMLTagRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	// [(text)] database cylinder: strip outer parens from captured label.
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = s[1 : len(s)-1]
	}
	return s
}

// parseFCNodeExpr extracts (id, label, kind) from a node expression like "A[My Label]".
func parseFCNodeExpr(s string) (id, label, kind string) {
	s = strings.TrimSpace(s)
	m := fcNodeRe.FindStringSubmatch(s)
	if m == nil {
		return s, s, "rect"
	}
	id = m[1]
	switch {
	case m[3] != "":
		label, kind = cleanFCLabel(m[3]), "rect"
	case m[4] != "":
		label, kind = cleanFCLabel(m[4]), "diamond"
	case m[5] != "":
		label, kind = cleanFCLabel(m[5]), "db"
	case m[6] != "":
		label, kind = cleanFCLabel(m[6]), "round"
	default:
		label, kind = id, "rect"
	}
	return
}

// sgAddNode adds a node to a subgraph, updating the label only when an explicit
// bracket declaration is provided (i.e. the label differs from the bare id).
func sgAddNode(sg *sgBlock, id, label, kind string) {
	if _, exists := sg.nodes[id]; !exists {
		sg.nodeOrder = append(sg.nodeOrder, id)
		sg.nodes[id] = fcNode{id: id, label: label, kind: kind}
	} else if label != id {
		sg.nodes[id] = fcNode{id: id, label: label, kind: kind}
	}
}

// parseLabeledEdge detects "A -->|label| B" or "A -- label --> B" patterns.
func parseLabeledEdge(line string) (from, to, label string, ok bool) {
	// A -- text --> B
	if i := strings.Index(line, " -- "); i >= 0 {
		rest := line[i+4:]
		if j := strings.Index(rest, " -->"); j >= 0 {
			return strings.TrimSpace(line[:i]),
				strings.TrimSpace(rest[j+4:]),
				strings.TrimSpace(rest[:j]),
				true
		}
	}
	// A -->|label| B  (or -.->, ==>, ---)
	for _, pre := range []string{"-.->", "==>", "-->", "---"} {
		marker := pre + "|"
		if i := strings.Index(line, marker); i >= 0 {
			rest := line[i+len(marker):]
			if j := strings.Index(rest, "|"); j >= 0 {
				return strings.TrimSpace(line[:i]),
					strings.TrimSpace(rest[j+1:]),
					rest[:j],
					true
			}
		}
	}
	return "", "", "", false
}

// parseLineEdges returns all edges declared on one flowchart line, along with
// the raw node expressions (for label extraction). Handles:
//   - labeled:  A -->|lbl| B
//   - chained:  A --> B --> C  (three or more tokens)
//   - plain:    A --> B
type rawEdge struct{ fromExpr, toExpr, label string }

func parseLineEdges(line string) []rawEdge {
	// Labeled edge takes priority (so we don't confuse it with a chained edge).
	if from, to, lbl, ok := parseLabeledEdge(line); ok {
		return []rawEdge{{fromExpr: from, toExpr: to, label: lbl}}
	}
	// Chained or plain edge.
	for _, sep := range []string{" -.-> ", " --> ", " --- ", " ==> "} {
		parts := strings.Split(line, sep)
		if len(parts) < 2 {
			continue
		}
		var edges []rawEdge
		for i := 0; i < len(parts)-1; i++ {
			edges = append(edges, rawEdge{
				fromExpr: strings.TrimSpace(parts[i]),
				toExpr:   strings.TrimSpace(parts[i+1]),
			})
		}
		return edges
	}
	return nil
}

// renderFlowchart is the top-level flowchart renderer.
func renderFlowchart(lines []string, width int) string {
	var subgraphs []*sgBlock
	var globalEdges []rawEdge
	// allNodes is a shared registry; subgraph nodes are also tracked per-sg.
	allNodes := map[string]fcNode{}

	var current *sgBlock   // nil = top level
	var sgStack []*sgBlock // stack for nesting (we handle one level deep)

	addToContext := func(sg *sgBlock, id, label, kind string) {
		if _, exists := allNodes[id]; !exists || label != id {
			allNodes[id] = fcNode{id: id, label: label, kind: kind}
		}
		if sg != nil {
			sgAddNode(sg, id, label, kind)
		}
	}

	for _, raw := range lines[1:] {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		kw := strings.ToLower(fields[0])

		// ── Subgraph start ────────────────────────────────────────────────────
		if kw == "subgraph" {
			m := fcSubgraphRe.FindStringSubmatch(line)
			var sg *sgBlock
			if m != nil {
				sg = newSGBlock(m[1], m[2])
			} else {
				sg = newSGBlock("sg", "")
			}
			subgraphs = append(subgraphs, sg)
			sgStack = append(sgStack, current)
			current = sg
			continue
		}

		// ── Subgraph end ──────────────────────────────────────────────────────
		if kw == "end" && len(sgStack) > 0 {
			current = sgStack[len(sgStack)-1]
			sgStack = sgStack[:len(sgStack)-1]
			continue
		}

		// ── Skip decorator keywords ───────────────────────────────────────────
		switch kw {
		case "style", "classDef", "classdef", "class", "click", "direction", "linkstyle":
			continue
		}

		// ── Edges ─────────────────────────────────────────────────────────────
		if edges := parseLineEdges(line); edges != nil {
			for _, e := range edges {
				fid, flbl, fkind := parseFCNodeExpr(e.fromExpr)
				tid, tlbl, tkind := parseFCNodeExpr(e.toExpr)
				addToContext(current, fid, flbl, fkind)
				addToContext(current, tid, tlbl, tkind)
				fe := fcEdge{from: fid, to: tid, label: e.label}
				if current != nil {
					current.edges = append(current.edges, fe)
				} else {
					globalEdges = append(globalEdges, rawEdge{
						fromExpr: fid, toExpr: tid, label: e.label,
					})
				}
				_ = tlbl
				_ = tkind
			}
			continue
		}

		// ── Standalone node declaration ───────────────────────────────────────
		if id, lbl, kind := parseFCNodeExpr(line); id != "" {
			addToContext(current, id, lbl, kind)
		}
	}

	if len(allNodes) == 0 {
		return ""
	}

	if len(subgraphs) > 0 {
		return renderSubgraphs(subgraphs, globalEdges, allNodes, width)
	}

	// No subgraphs: flatten and try simple chain.
	var nodeOrder []string
	seen := map[string]bool{}
	for _, e := range globalEdges {
		for _, id := range []string{e.fromExpr, e.toExpr} {
			if !seen[id] {
				seen[id] = true
				nodeOrder = append(nodeOrder, id)
			}
		}
	}
	for id := range allNodes {
		if !seen[id] {
			seen[id] = true
			nodeOrder = append(nodeOrder, id)
		}
	}

	// Check for branching/merging.
	outCount, inCount := map[string]int{}, map[string]int{}
	var flatEdges []fcEdge
	for _, re := range globalEdges {
		flatEdges = append(flatEdges, fcEdge{from: re.fromExpr, to: re.toExpr, label: re.label})
		outCount[re.fromExpr]++
		inCount[re.toExpr]++
	}
	for _, c := range outCount {
		if c > 1 {
			return ""
		}
	}
	for _, c := range inCount {
		if c > 1 {
			return ""
		}
	}

	return renderChain(nodeOrder, allNodes, flatEdges, width)
}

// ── Subgraph rendering ────────────────────────────────────────────────────────

func renderSubgraphs(sgs []*sgBlock, globalEdges []rawEdge, allNodes map[string]fcNode, width int) string {
	const outerIndent = 2
	outerBW := width - outerIndent // box width including borders
	if outerBW < 20 {
		outerBW = 20
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(diagTitleANSI + "  ⬡ Flowchart" + ansiReset + "\n")

	for _, sg := range sgs {
		sb.WriteString("\n")
		renderSGBox(&sb, sg, outerIndent, outerBW)
	}

	// Cross-subgraph / global connections.
	if len(globalEdges) > 0 {
		sb.WriteString("\n")
		sb.WriteString(diagLineANSI + strings.Repeat(" ", outerIndent) + "Connections:" + ansiReset + "\n")
		for _, e := range globalEdges {
			fromLbl := e.fromExpr
			if n, ok := allNodes[e.fromExpr]; ok {
				fromLbl = n.label
			}
			toLbl := e.toExpr
			if n, ok := allNodes[e.toExpr]; ok {
				toLbl = n.label
			}
			arrow := "──▶"
			conn := strings.Repeat(" ", outerIndent+2) + fromLbl + "  " + arrow + "  " + toLbl
			if e.label != "" {
				conn += "   (" + e.label + ")"
			}
			sb.WriteString(diagLineANSI + conn + ansiReset + "\n")
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// renderSGBox draws a subgraph as a labelled bordered box containing its chain.
func renderSGBox(sb *strings.Builder, sg *sgBlock, outerIndent, outerBW int) {
	innerW := outerBW - 2 // characters between the │ border columns

	// emitBorder draws a full-width border row.
	emitBorder := func(left, fill, right rune, titleText string) {
		row := mkRow(outerIndent + outerBW)
		setR(row, outerIndent, left)
		for x := outerIndent + 1; x < outerIndent+outerBW-1; x++ {
			setR(row, x, fill)
		}
		if titleText != "" {
			setS(row, outerIndent+2, titleText)
		}
		setR(row, outerIndent+outerBW-1, right)
		emitRow(sb, row, diagBoxANSI)
	}

	// emitContent wraps a content row (plain runes, innerW wide) in │…│.
	emitContent := func(contentRunes []rune, contentColor string) {
		vw := len(contentRunes)
		pad := innerW - vw
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(strings.Repeat(" ", outerIndent))
		sb.WriteString(diagBoxANSI + "│" + ansiReset)
		sb.WriteString(contentColor + string(contentRunes) + strings.Repeat(" ", pad) + ansiReset)
		sb.WriteString(diagBoxANSI + "│" + ansiReset + "\n")
	}

	emitBlankLine := func() { emitContent(mkRow(0), diagBoxANSI) }

	// ┌──────────────────────────────────────────────┐
	emitBorder('┌', '─', '┐', "")
	// │  SubgraphLabel                               │
	emitContent([]rune("  "+sg.label), diagBoxANSI)
	// ├──────────────────────────────────────────────┤
	emitBorder('├', '─', '┤', "")
	emitBlankLine()

	// Build and render the chain inside the box.
	chainRows := buildChainRows(sg, innerW)
	for _, cr := range chainRows {
		emitContent(cr.runes, cr.color)
	}
	if len(chainRows) == 0 {
		// Fallback: list nodes.
		for _, id := range sg.nodeOrder {
			n := sg.nodes[id]
			emitContent([]rune("  • "+n.label), diagBoxANSI)
		}
	}

	emitBlankLine()
	// └──────────────────────────────────────────────┘
	emitBorder('└', '─', '┘', "")
}

// chainRow is one rendered line of a chain, with its ANSI color.
type chainRow struct {
	runes []rune
	color string
}

// buildChainRows converts a subgraph's node chain into a slice of plain rune
// rows (no ANSI) ready to be wrapped inside a box. Each row carries a color tag.
func buildChainRows(sg *sgBlock, innerW int) []chainRow {
	// Follow the chain from root.
	inCount := map[string]int{}
	nextOf := map[string]string{}
	edgeLbl := map[string]string{}
	for _, e := range sg.edges {
		inCount[e.to]++
		nextOf[e.from] = e.to
		edgeLbl[e.from] = e.label
	}

	root := ""
	for _, id := range sg.nodeOrder {
		if inCount[id] == 0 {
			root = id
			break
		}
	}
	if root == "" && len(sg.nodeOrder) > 0 {
		root = sg.nodeOrder[0]
	}

	var chain []string
	visited := map[string]bool{}
	for cur := root; cur != "" && !visited[cur]; cur = nextOf[cur] {
		chain = append(chain, cur)
		visited[cur] = true
	}
	if len(chain) == 0 {
		return nil
	}

	// Size the inner node boxes.
	maxLbl := 0
	for _, id := range chain {
		if l := utf8.RuneCountInString(sg.nodes[id].label); l > maxLbl {
			maxLbl = l
		}
	}
	const innerIndent = 2
	boxW := maxLbl + 4
	if boxW < 8 {
		boxW = 8
	}
	if boxW%2 != 0 {
		boxW++
	}
	maxBoxW := innerW - innerIndent - 2
	if maxBoxW > 8 && boxW > maxBoxW {
		boxW = maxBoxW
	}
	half := boxW / 2
	cx := innerIndent + half

	mkInner := func() []rune { return mkRow(innerW) }

	var rows []chainRow
	box := func(r []rune) { rows = append(rows, chainRow{runes: r, color: diagBoxANSI}) }
	line := func(r []rune) { rows = append(rows, chainRow{runes: r, color: diagLineANSI}) }

	for i, id := range chain {
		n := sg.nodes[id]
		isLast := i == len(chain)-1

		lbl := n.label
		lw := utf8.RuneCountInString(lbl)
		inner := boxW - 2
		if lw > inner {
			rr := []rune(lbl)
			lbl = string(rr[:inner-1]) + "…"
			lw = inner
		}
		lpad := (inner - lw) / 2

		if n.kind == "diamond" {
			top := mkInner()
			setR(top, innerIndent, '╔')
			for x := innerIndent + 1; x < innerIndent+boxW-1; x++ {
				setR(top, x, '═')
			}
			setR(top, innerIndent+boxW-1, '╗')
			box(top)

			mid := mkInner()
			setR(mid, innerIndent, '║')
			setR(mid, innerIndent+boxW-1, '║')
			setS(mid, innerIndent+1+lpad, lbl)
			box(mid)

			bot := mkInner()
			setR(bot, innerIndent, '╚')
			for x := innerIndent + 1; x < cx; x++ {
				setR(bot, x, '═')
			}
			if isLast {
				setR(bot, cx, '═')
			} else {
				setR(bot, cx, '╦')
			}
			for x := cx + 1; x < innerIndent+boxW-1; x++ {
				setR(bot, x, '═')
			}
			setR(bot, innerIndent+boxW-1, '╝')
			box(bot)
		} else {
			top := mkInner()
			setR(top, innerIndent, '┌')
			for x := innerIndent + 1; x < innerIndent+boxW-1; x++ {
				setR(top, x, '─')
			}
			setR(top, innerIndent+boxW-1, '┐')
			box(top)

			mid := mkInner()
			setR(mid, innerIndent, '│')
			setR(mid, innerIndent+boxW-1, '│')
			setS(mid, innerIndent+1+lpad, lbl)
			box(mid)

			bot := mkInner()
			setR(bot, innerIndent, '└')
			for x := innerIndent + 1; x < cx; x++ {
				setR(bot, x, '─')
			}
			if isLast {
				setR(bot, cx, '─')
			} else {
				setR(bot, cx, '┬')
			}
			for x := cx + 1; x < innerIndent+boxW-1; x++ {
				setR(bot, x, '─')
			}
			setR(bot, innerIndent+boxW-1, '┘')
			box(bot)
		}

		if !isLast {
			lbl := edgeLbl[id]

			r := mkInner()
			setR(r, cx, '│')
			line(r)

			if lbl != "" {
				r = mkInner()
				setR(r, cx, '├')
				setS(r, cx+1, "─ "+lbl)
				line(r)

				r = mkInner()
				setR(r, cx, '│')
				line(r)
			}

			r = mkInner()
			setR(r, cx, '▼')
			line(r)
		}
	}
	return rows
}

// ── Simple chain (no subgraphs) ───────────────────────────────────────────────

func renderChain(order []string, nodes map[string]fcNode, edges []fcEdge, width int) string {
	inCount := map[string]int{}
	nextOf := map[string]string{}
	edgeLbl := map[string]string{}
	for _, e := range edges {
		inCount[e.to]++
		nextOf[e.from] = e.to
		edgeLbl[e.from] = e.label
	}

	root := ""
	for _, id := range order {
		if inCount[id] == 0 {
			root = id
			break
		}
	}
	if root == "" && len(order) > 0 {
		root = order[0]
	}

	var chain []string
	visited := map[string]bool{}
	for cur := root; cur != "" && !visited[cur]; cur = nextOf[cur] {
		chain = append(chain, cur)
		visited[cur] = true
	}
	if len(chain) == 0 {
		return ""
	}

	maxLbl := 0
	for _, id := range chain {
		if l := utf8.RuneCountInString(nodes[id].label); l > maxLbl {
			maxLbl = l
		}
	}
	boxW := maxLbl + 4
	if boxW < 8 {
		boxW = 8
	}
	if boxW%2 != 0 {
		boxW++
	}
	if boxW > width-4 {
		boxW = width - 4
	}

	const indent = 2
	half := boxW / 2
	cx := indent + half
	rowW := indent + boxW + 2

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(diagTitleANSI + "  ⬡ Flowchart" + ansiReset + "\n\n")

	for i, id := range chain {
		n := nodes[id]
		lbl := n.label
		lw := utf8.RuneCountInString(lbl)
		inner := boxW - 2
		lpad := (inner - lw) / 2
		isLast := i == len(chain)-1

		if n.kind == "diamond" {
			top := mkRow(rowW)
			setR(top, indent, '╔')
			for x := indent + 1; x < indent+boxW-1; x++ {
				setR(top, x, '═')
			}
			setR(top, indent+boxW-1, '╗')
			emitRow(&sb, top, diagBoxANSI)

			mid := mkRow(rowW)
			setR(mid, indent, '║')
			setR(mid, indent+boxW-1, '║')
			setS(mid, indent+1+lpad, lbl)
			emitRow(&sb, mid, diagBoxANSI)

			bot := mkRow(rowW)
			setR(bot, indent, '╚')
			for x := indent + 1; x < cx; x++ {
				setR(bot, x, '═')
			}
			if isLast {
				setR(bot, cx, '═')
			} else {
				setR(bot, cx, '╦')
			}
			for x := cx + 1; x < indent+boxW-1; x++ {
				setR(bot, x, '═')
			}
			setR(bot, indent+boxW-1, '╝')
			emitRow(&sb, bot, diagBoxANSI)
		} else {
			top := mkRow(rowW)
			setR(top, indent, '┌')
			for x := indent + 1; x < indent+boxW-1; x++ {
				setR(top, x, '─')
			}
			setR(top, indent+boxW-1, '┐')
			emitRow(&sb, top, diagBoxANSI)

			mid := mkRow(rowW)
			setR(mid, indent, '│')
			setR(mid, indent+boxW-1, '│')
			setS(mid, indent+1+lpad, lbl)
			emitRow(&sb, mid, diagBoxANSI)

			bot := mkRow(rowW)
			setR(bot, indent, '└')
			for x := indent + 1; x < cx; x++ {
				setR(bot, x, '─')
			}
			if isLast {
				setR(bot, cx, '─')
			} else {
				setR(bot, cx, '┬')
			}
			for x := cx + 1; x < indent+boxW-1; x++ {
				setR(bot, x, '─')
			}
			setR(bot, indent+boxW-1, '┘')
			emitRow(&sb, bot, diagBoxANSI)
		}

		if !isLast {
			lbl := edgeLbl[id]

			r := mkRow(rowW)
			setR(r, cx, '│')
			emitRow(&sb, r, diagLineANSI)

			if lbl != "" {
				r = mkRow(cx + utf8.RuneCountInString(lbl) + 4)
				setR(r, cx, '├')
				setS(r, cx+1, "─ "+lbl)
				emitRow(&sb, r, diagLineANSI)

				r = mkRow(rowW)
				setR(r, cx, '│')
				emitRow(&sb, r, diagLineANSI)
			}

			r = mkRow(rowW)
			setR(r, cx, '▼')
			emitRow(&sb, r, diagLineANSI)
		}
	}

	sb.WriteString("\n")
	return sb.String()
}
