	"github.com/git-lfs/git-lfs/filepathfilter"
	scanner := newLogScanner(dir, log)
	if len(includePaths)+len(excludePaths) > 0 {
		scanner.Filter = filepathfilter.New(includePaths, excludePaths)
	}
	for scanner.Scan() {
		if p := scanner.Pointer(); p != nil {
			results <- p
}
// logScanner parses log output formatted as per logLfsSearchArgs & returns
// pointers.
type logScanner struct {
	// Filter will ensure file paths matching the include patterns, or not matchin
	// the exclude patterns are skipped.
	Filter *filepathfilter.Filter

	s       *bufio.Scanner
	dir     LogDiffDirection
	pointer *WrappedPointer

	pointerData         *bytes.Buffer
	currentFilename     string
	currentFileIncluded bool

	commitHeaderRegex    *regexp.Regexp
	fileHeaderRegex      *regexp.Regexp
	fileMergeHeaderRegex *regexp.Regexp
	pointerDataRegex     *regexp.Regexp
}

// dir: whether to include results from + or - diffs
// r: a stream of output from git log with at least logLfsSearchArgs specified
func newLogScanner(dir LogDiffDirection, r io.Reader) *logScanner {
	return &logScanner{
		s:                   bufio.NewScanner(r),
		dir:                 dir,
		pointerData:         &bytes.Buffer{},
		currentFileIncluded: true,

		// no need to compile these regexes on every `git-lfs` call, just ones that
		// use the scanner.
		commitHeaderRegex:    regexp.MustCompile(`^lfs-commit-sha: ([A-Fa-f0-9]{40})(?: ([A-Fa-f0-9]{40}))*`),
		fileHeaderRegex:      regexp.MustCompile(`diff --git a\/(.+?)\s+b\/(.+)`),
		fileMergeHeaderRegex: regexp.MustCompile(`diff --cc (.+)`),
		pointerDataRegex:     regexp.MustCompile(`^([\+\- ])(version https://git-lfs|oid sha256|size|ext-).*$`),
	}
}

func (s *logScanner) Pointer() *WrappedPointer {
	return s.pointer
}

func (s *logScanner) Err() error {
	return s.s.Err()
}

func (s *logScanner) Scan() bool {
	s.pointer = nil
	p, canScan := s.scan()
	s.pointer = p
	return canScan
}

// Utility func used at several points below (keep in narrow scope)
func (s *logScanner) finishLastPointer() *WrappedPointer {
	if s.pointerData.Len() == 0 || !s.currentFileIncluded {
		return nil
	}

	p, err := DecodePointer(s.pointerData)
	s.pointerData.Reset()

	if err == nil {
		return &WrappedPointer{Name: s.currentFilename, Pointer: p}
	} else {
		tracerx.Printf("Unable to parse pointer from log: %v", err)
		return nil
	}
}

// For each commit we'll get something like this:
/*
	lfs-commit-sha: 60fde3d23553e10a55e2a32ed18c20f65edd91e7 e2eaf1c10b57da7b98eb5d722ec5912ddeb53ea1

	diff --git a/1D_Noise.png b/1D_Noise.png
	new file mode 100644
	index 0000000..2622b4a
	--- /dev/null
	+++ b/1D_Noise.png
	@@ -0,0 +1,3 @@
	+version https://git-lfs.github.com/spec/v1
	+oid sha256:f5d84da40ab1f6aa28df2b2bf1ade2cdcd4397133f903c12b4106641b10e1ed6
	+size 1289
*/
// There can be multiple diffs per commit (multiple binaries)
// Also when a binary is changed the diff will include a '-' line for the old SHA
func (s *logScanner) scan() (*WrappedPointer, bool) {
	for s.s.Scan() {
		line := s.s.Text()

		if match := s.commitHeaderRegex.FindStringSubmatch(line); match != nil {
			if p := s.finishLastPointer(); p != nil {
				return p, true
			}
		} else if match := s.fileHeaderRegex.FindStringSubmatch(line); match != nil {
			p := s.finishLastPointer()

			if s.dir == LogDiffAdditions {
				s.setFilename(match[2])
				s.setFilename(match[1])

			if p != nil {
				return p, true
			}
		} else if match := s.fileMergeHeaderRegex.FindStringSubmatch(line); match != nil {
			p := s.finishLastPointer()

			s.setFilename(match[1])

			if p != nil {
				return p, true
			}
		} else if s.currentFileIncluded {
			if match := s.pointerDataRegex.FindStringSubmatch(line); match != nil {

				if LogDiffDirection(changeType) == s.dir || changeType == ' ' {
					s.pointerData.WriteString(line[1:])
					s.pointerData.WriteString("\n") // newline was stripped off by scanner

	if p := s.finishLastPointer(); p != nil {
		return p, true
	}

	return nil, false
}

func (s *logScanner) setFilename(name string) {
	s.currentFilename = name
	s.currentFileIncluded = s.Filter.Allows(name)