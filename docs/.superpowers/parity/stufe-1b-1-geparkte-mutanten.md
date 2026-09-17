# Stufe 1b-1 — die geparkten Überlebenden der Mutationsrunde

**Bezug:** [Paritätsliste Stufe 1b-1](stufe-1b-1.md), Abschnitt „Geparkt (Ruling
R19)". Die Runde der Stufe hat `internal/brain/guard` und `internal/hooks`
mitgefahren, aber nicht bewertet: beides sind Pakete der Stufe 1a, deren Tests
diese Stufe nicht geschrieben hat. Die Zeilen standen bis hierher nur in
`%TEMP%\mutants-1b1\`, also flüchtig; wer die beiden Pakete nachholt, braucht
die Runde mit ihnen nicht zu wiederholen.

**Stand des Codes:** gelaufen am 2026-09-16 mit `bin/loomux.exe dev mutants
<paket>`, 8 Arbeiter, gegen Commit `d50c2cd` („Count a mutant whose suite exited
zero as a survivor"). **Die Datei- und Zeilenangaben gelten für diesen Commit**
und wandern mit jeder späteren Änderung; wer sie auflöst, liest sie dort
(`git show d50c2cd:internal/brain/guard/guard.go`).

## `internal/brain/guard` — 17 Überlebende von 771 Mutanten

```text
[47/771] SURVIVED  (a1) guard.go:213  if err != nil {  ->  if false {
[104/771] SURVIVED  (a2) guard.go:327  if rootErr != nil || hereErr != nil {  ->  if rootErr != nil {
[119/771] SURVIVED  (a1) guard.go:407  if len(targets) == 0 {  ->  if false {
[171/771] SURVIVED  (a1) guard.go:506  if len(outside) == 0 {  ->  if false {
[175/771] SURVIVED  (a1) guard.go:520  if len(outside) == 0 {  ->  if false {
[189/771] SURVIVED  (a1) guard.go:540  if scratch != "" {  ->  if true {
[409/771] SURVIVED  (a3) memory.go:101  if len(parts) > len(target) || !slices.Equal(target[:len(parts)], parts) {  ->  if len(parts) >= len(target) || !slices.Equal(target[:len(parts)], parts) {
[446/771] SURVIVED  (a1) path.go:131  if tail == "" {  ->  if false {
[487/771] SURVIVED  (a3) path.go:303  return r < 0x80 && os.IsPathSeparator(byte(r))  ->  return r <= 0x80 && os.IsPathSeparator(byte(r))
[495/771] SURVIVED  (a1) path.go:424  if err != nil {  ->  if false {
[506/771] SURVIVED  (a1) path.go:444  if err != nil {  ->  if false {
[540/771] SURVIVED  (a3) path_windows.go:75  if int(length) <= len(buffer) {  ->  if int(length) < len(buffer) {
[712/771] SURVIVED  (a1) scratchpad.go:33  if base == "" {  ->  if false {
[729/771] SURVIVED  (a1) worktree.go:52  if common == "" {  ->  if false {
[742/771] SURVIVED  (a1) worktree.go:93  if info, err := os.Lstat(dotGit); err != nil || !info.Mode().IsRegular() {  ->  if false {
[748/771] SURVIVED  (a1) worktree.go:97  if admin == "" {  ->  if false {
[755/771] SURVIVED  (a2) worktree.go:101  if back == "" || !pathsEqual(back, resolvedOrEmpty(dotGit)) {  ->  if !pathsEqual(back, resolvedOrEmpty(dotGit)) {
```

## `internal/hooks` — 109 Überlebende von 706 Mutanten

```text
[2/706] SURVIVED  (a1) guard.go:78  if pattern == path {  ->  if false {
[5/706] SURVIVED  (a1) guard.go:82  if strings.HasSuffix(pattern, "/**") {  ->  if true {
[6/706] SURVIVED  (a1) guard.go:82  if strings.HasSuffix(pattern, "/**") {  ->  if false {
[7/706] SURVIVED  (a4) guard.go:82  if strings.HasSuffix(pattern, "/**") {  ->  if !(strings.HasSuffix(pattern, "/**")) {
[11/706] SURVIVED  (a2) guard.go:84  if path == prefix || strings.HasPrefix(path, prefix+"/") {  ->  if path == prefix {
[12/706] SURVIVED  (a2) guard.go:84  if path == prefix || strings.HasPrefix(path, prefix+"/") {  ->  if strings.HasPrefix(path, prefix+"/") {
[14/706] SURVIVED  (a1) guard.go:88  if strings.Contains(pattern, "/") {  ->  if true {
[18/706] SURVIVED  (a1) guard.go:111  if !filepath.IsAbs(raw) {  ->  if false {
[23/706] SURVIVED  (a2) guard.go:115  if err != nil || strings.HasPrefix(rel, "..") {  ->  if err != nil {
[36/706] SURVIVED  (a1) guard.go:148  if commandTools[tool] {  ->  if true {
[46/706] SURVIVED  (a1) hook_session_start.go:39  if err != nil {  ->  if false {
[70/706] SURVIVED  (a1) hook_session_start.go:110  if err != nil {  ->  if false {
[74/706] SURVIVED  (a1) hook_session_start.go:114  if err != nil {  ->  if false {
[81/706] SURVIVED  (a2) hook_session_start.go:118  if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {  ->  if strings.HasPrefix(rel, "..") {
[92/706] SURVIVED  (a2) hook_session_start.go:126  if err != nil || !info.ModTime().Before(newest) {  ->  if !info.ModTime().Before(newest) {
[104/706] SURVIVED  (a2) hook_session_start.go:153  if errors.Is(err, fs.ErrNotExist) && path == filepath.Join(root, dir) {  ->  if errors.Is(err, fs.ErrNotExist) {
[114/706] SURVIVED  (a2) hook_session_start.go:159  if entry.IsDir() || filepath.Ext(path) != ".go" {  ->  if entry.IsDir() {
[115/706] SURVIVED  (a2) hook_session_start.go:159  if entry.IsDir() || filepath.Ext(path) != ".go" {  ->  if filepath.Ext(path) != ".go" {
[122/706] SURVIVED  (a1) hook_session_start.go:169  if err != nil {  ->  if false {
[148/706] SURVIVED  (a2) post_edit.go:192  if !ok || rawPath == "" {  ->  if !ok {
[156/706] SURVIVED  (a1) post_edit.go:200  if explicitIgnoredExtensions[ext] {  ->  if false {
[172/706] SURVIVED  (a1) post_edit.go:236  if out == "" {  ->  if true {
[173/706] SURVIVED  (a1) post_edit.go:236  if out == "" {  ->  if false {
[174/706] SURVIVED  (a4) post_edit.go:236  if out == "" {  ->  if !(out == "") {
[175/706] SURVIVED  (a3) post_edit.go:236  if out == "" {  ->  if out != "" {
[188/706] SURVIVED  (a1) post_edit.go:316  if !hasTarget || targetPath == "" {  ->  if false {
[190/706] SURVIVED  (a2) post_edit.go:316  if !hasTarget || targetPath == "" {  ->  if !hasTarget {
[191/706] SURVIVED  (a2) post_edit.go:316  if !hasTarget || targetPath == "" {  ->  if targetPath == "" {
[211/706] SURVIVED  (a1) post_edit.go:342  if hasTarget && targetStack != "" {  ->  if false {
[213/706] SURVIVED  (a2) post_edit.go:342  if hasTarget && targetStack != "" {  ->  if hasTarget {
[214/706] SURVIVED  (a2) post_edit.go:342  if hasTarget && targetStack != "" {  ->  if targetStack != "" {
[215/706] SURVIVED  (a3) post_edit.go:342  if hasTarget && targetStack != "" {  ->  if hasTarget && targetStack == "" {
[232/706] SURVIVED  (a2) post_edit.go:364  if hasTarget && targetPath != "" {  ->  if hasTarget {
[233/706] SURVIVED  (a2) post_edit.go:364  if hasTarget && targetPath != "" {  ->  if targetPath != "" {
[238/706] SURVIVED  (a1) post_edit.go:371  if hasTarget && targetPath != "" {  ->  if true {
[239/706] SURVIVED  (a1) post_edit.go:371  if hasTarget && targetPath != "" {  ->  if false {
[240/706] SURVIVED  (a4) post_edit.go:371  if hasTarget && targetPath != "" {  ->  if !(hasTarget && targetPath != "") {
[241/706] SURVIVED  (a2) post_edit.go:371  if hasTarget && targetPath != "" {  ->  if hasTarget {
[242/706] SURVIVED  (a2) post_edit.go:371  if hasTarget && targetPath != "" {  ->  if targetPath != "" {
[243/706] SURVIVED  (a3) post_edit.go:371  if hasTarget && targetPath != "" {  ->  if hasTarget && targetPath == "" {
[251/706] SURVIVED  (a1) post_edit.go:379  if hasTarget && targetPath != "" {  ->  if true {
[254/706] SURVIVED  (a2) post_edit.go:379  if hasTarget && targetPath != "" {  ->  if hasTarget {
[255/706] SURVIVED  (a2) post_edit.go:379  if hasTarget && targetPath != "" {  ->  if targetPath != "" {
[260/706] SURVIVED  (a2) post_edit.go:391  if hasTarget && targetPath != "" {  ->  if hasTarget {
[261/706] SURVIVED  (a2) post_edit.go:391  if hasTarget && targetPath != "" {  ->  if targetPath != "" {
[283/706] SURVIVED  (a2) post_edit.go:413  if targetDir != "" && hasTarget && targetPath != "" {  ->  if targetDir != "" {
[295/706] SURVIVED  (a2) post_edit.go:422  if hasTarget && targetPath != "" {  ->  if hasTarget {
[296/706] SURVIVED  (a2) post_edit.go:422  if hasTarget && targetPath != "" {  ->  if targetPath != "" {
[304/706] SURVIVED  (a2) post_edit.go:429  if hasTarget && targetPath != "" {  ->  if hasTarget {
[305/706] SURVIVED  (a2) post_edit.go:429  if hasTarget && targetPath != "" {  ->  if targetPath != "" {
[313/706] SURVIVED  (a2) post_edit.go:436  if hasTarget && targetPath != "" {  ->  if hasTarget {
[314/706] SURVIVED  (a2) post_edit.go:436  if hasTarget && targetPath != "" {  ->  if targetPath != "" {
[329/706] SURVIVED  (a1) post_edit.go:468  if !filepath.IsAbs(page) {  ->  if true {
[337/706] SURVIVED  (a1) post_edit.go:491  if area == "" {  ->  if false {
[340/706] SURVIVED  (a1) post_edit.go:496  if strings.HasPrefix(norm, prefix) {  ->  if true {
[360/706] SURVIVED  (a1) post_edit.go:546  if err != nil || layout == "" {  ->  if false {
[362/706] SURVIVED  (a2) post_edit.go:546  if err != nil || layout == "" {  ->  if err != nil {
[363/706] SURVIVED  (a2) post_edit.go:546  if err != nil || layout == "" {  ->  if layout == "" {
[371/706] SURVIVED  (a2) post_edit.go:555  if slices.Contains(stacks, "wiki") || !declaresWikiLayout(projectRoot) {  ->  if !declaresWikiLayout(projectRoot) {
[377/706] SURVIVED  (a1) post_edit.go:575  if root != "" {  ->  if true {
[391/706] SURVIVED  (a1) pretool.go:26  if err != nil {  ->  if false {
[395/706] SURVIVED  (a1) pretool.go:30  if err != nil {  ->  if false {
[411/706] SURVIVED  (a1) pretool.go:52  if err != nil {  ->  if false {
[425/706] SURVIVED  (a2) status.go:63  if strings.Contains(cmd, "loomux") && strings.Contains(cmd, "pre-tool-use") {  ->  if strings.Contains(cmd, "loomux") {
[426/706] SURVIVED  (a2) status.go:63  if strings.Contains(cmd, "loomux") && strings.Contains(cmd, "pre-tool-use") {  ->  if strings.Contains(cmd, "pre-tool-use") {
[428/706] SURVIVED  (a1) status.go:66  if strings.Contains(cmd, "loomux") && strings.Contains(cmd, "post-tool-use") {  ->  if false {
[431/706] SURVIVED  (a2) status.go:66  if strings.Contains(cmd, "loomux") && strings.Contains(cmd, "post-tool-use") {  ->  if strings.Contains(cmd, "post-tool-use") {
[437/706] SURVIVED  (a4) status.go:88  if err != nil {  ->  if !(err != nil) {
[438/706] SURVIVED  (a3) status.go:88  if err != nil {  ->  if err == nil {
[446/706] SURVIVED  (a1) status.go:136  if hasStack("python") {  ->  if true {
[452/706] SURVIVED  (a1) status.go:138  if hasStack("uv") {  ->  if true {
[455/706] SURVIVED  (a1) status.go:153  if hasStack("gdscript") {  ->  if true {
[458/706] SURVIVED  (a1) status.go:156  if hasStack("cpp") {  ->  if true {
[459/706] SURVIVED  (a1) status.go:156  if hasStack("cpp") {  ->  if false {
[460/706] SURVIVED  (a4) status.go:156  if hasStack("cpp") {  ->  if !(hasStack("cpp")) {
[461/706] SURVIVED  (a1) status.go:160  if hasStack("typescript") {  ->  if true {
[464/706] SURVIVED  (a1) status.go:164  if hasStack("vue") {  ->  if true {
[467/706] SURVIVED  (a1) status.go:167  if hasStack("svelte") {  ->  if true {
[470/706] SURVIVED  (a1) status.go:170  if hasStack("css") {  ->  if true {
[473/706] SURVIVED  (a1) status.go:173  if hasStack("html") {  ->  if true {
[476/706] SURVIVED  (a1) status.go:176  if hasStack("shell") {  ->  if true {
[479/706] SURVIVED  (a1) status.go:179  if hasStack("sql") {  ->  if true {
[480/706] SURVIVED  (a1) status.go:179  if hasStack("sql") {  ->  if false {
[481/706] SURVIVED  (a4) status.go:179  if hasStack("sql") {  ->  if !(hasStack("sql")) {
[482/706] SURVIVED  (a1) status.go:182  if hasStack("rust") {  ->  if true {
[485/706] SURVIVED  (a1) status.go:186  if hasStack("go") {  ->  if true {
[488/706] SURVIVED  (a1) status.go:187  if hasStack("golangci-lint") {  ->  if true {
[489/706] SURVIVED  (a1) status.go:187  if hasStack("golangci-lint") {  ->  if false {
[490/706] SURVIVED  (a4) status.go:187  if hasStack("golangci-lint") {  ->  if !(hasStack("golangci-lint")) {
[491/706] SURVIVED  (a1) status.go:194  if hasStack("wiki") {  ->  if true {
[494/706] SURVIVED  (a1) status.go:203  if hasStack("wiki") {  ->  if true {
[495/706] SURVIVED  (a1) status.go:203  if hasStack("wiki") {  ->  if false {
[496/706] SURVIVED  (a4) status.go:203  if hasStack("wiki") {  ->  if !(hasStack("wiki")) {
[512/706] SURVIVED  (a2) status.go:260  if cmd.run != nil || tool == "" || seen[tool] {  ->  if seen[tool] {
[524/706] SURVIVED  (a1) worktree.go:42  if err != nil {  ->  if false {
[532/706] SURVIVED  (a1) worktree.go:55  if len(mirrored) == 0 {  ->  if false {
[535/706] SURVIVED  (a1) worktree.go:60  if topology.IsWorktree(root) {  ->  if true {
[554/706] SURVIVED  (a1) worktree.go:122  if err != nil {  ->  if false {
[558/706] SURVIVED  (a1) worktree.go:128  if !topology.IsWorktree(root) {  ->  if false {
[565/706] SURVIVED  (a1) worktree.go:136  if len(mirrored) == 0 {  ->  if false {
[585/706] SURVIVED  (a1) worktree.go:185  if err != nil {  ->  if false {
[592/706] SURVIVED  (a1) worktree.go:197  if worktree == "" {  ->  if false {
[625/706] SURVIVED  (a2) worktree.go:293  if target == "" || !leadsInto(target, main) {  ->  if !leadsInto(target, main) {
[630/706] SURVIVED  (a3) worktree.go:296  if err := junction.Remove(path); err != nil {  ->  if err := junction.Remove(path); err == nil {
[664/706] SURVIVED  (a1) worktree.go:385  if target == "" {  ->  if false {
[673/706] SURVIVED  (a3) worktree.go:392  if err := junction.Remove(path); err != nil {  ->  if err := junction.Remove(path); err == nil {
[687/706] SURVIVED  (a2) worktree.go:475  if err != nil || !info.Mode().IsDir() {  ->  if !info.Mode().IsDir() {
[700/706] SURVIVED  (a1) worktree.go:527  if err != nil {  ->  if false {
[704/706] SURVIVED  (a1) worktree.go:531  if err != nil {  ->  if false {
```
