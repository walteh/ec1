package termlog

import (
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

var knownNestedProjects = []string{
	"github.com/moby/sys",
	"gitlab.com/tozd/go",
}

var currentMainGoPackage string

func init() {
	// zerolog.TimeFieldFormat = time.RFC3339Nano
	// zerolog.CallerMarshalFunc = ZeroLogCallerMarshalFunc

	inf, ok := debug.ReadBuildInfo()
	if ok {
		currentMainGoPackage = inf.Main.Path
		// for _, dep := range inf.Deps {
		// 	pkgMap[dep.Path] = dep
		// }
	}

}

const (
	githubIcon       = "\ue709 "
	githubIconSquare = "\uf092 "
	gitIcon          = "\ue725 "
	gitFolderIcon    = "\ue702 "
	goIcon           = "\ue65e "
	goXIcon          = "\ue702 "
	goStandardIcon   = "\uf07ef "
	gitlabIcon       = "\ue7eb "
	externalIcon     = "\uf14c "
)

func hyperlink(link, renderedText string) string {
	// OSC 8 sequence around styled text
	start := "\x1b]8;;" + link + "\x07"
	end := "\x1b]8;;\x07"

	return start + renderedText + end
}

type EnhancedSource struct {
	ptr             uintptr
	rawFunc         string
	rawFilePath     string
	rawFileLine     int
	enhancedFunc    string
	enhancedPkg     string
	enhancedProject string
	enhancedFullPkg string
}

func (e *EnhancedSource) Render(styles *Styles, render renderFunc, hyperlink HyperlinkFunc) string {

	pkgNoProject := strings.TrimPrefix(e.enhancedFullPkg, e.enhancedProject+"/")
	if e.enhancedProject == e.enhancedFullPkg {
		pkgNoProject = ""
	}

	var isCurrentMainGoPackage bool

	if e.enhancedProject == currentMainGoPackage {
		isCurrentMainGoPackage = true
	}

	var projIcon string

	// pkg = filepath.Base(pkg)
	filePath := render(styles.Caller.File, FileNameOfPath(e.rawFilePath))
	num := render(styles.Caller.Line, fmt.Sprintf("%d", e.rawFileLine))
	sep := render(styles.Caller.Sep, ":")

	if !isCurrentMainGoPackage {
		splt := strings.Split(e.enhancedProject, "/")
		first := splt[0]
		// var lasts []string
		// if len(splt) > 1 {
		// 	lasts = strings.Split(e.enhancedProject, "/")[1:]
		// }

		if first == "github.com" {
			projIcon = githubIcon
		} else if first == "gitlab.com" {
			projIcon = gitlabIcon
		} else if !strings.Contains(first, ".") {
			projIcon = goIcon
		} else {
			projIcon = externalIcon
		}
	} else {
		projIcon = gitIcon
	}

	pkgsplt := strings.Split(pkgNoProject, "/")
	// last := pkgsplt[len(pkgsplt)-1]
	pkg := render(styles.Caller.Pkg, strings.Join(pkgsplt, "/")) + sep
	if pkgNoProject == "" {
		pkg = ""
	}

	// var pkg string
	// if len(pkgsplt) > 1 {
	// 	pkg = styles.Caller.Pkg.Render(strings.Join(pkgsplt[:len(pkgsplt)-2], "/") + "/")
	// 	pkg += styles.Caller.Pkg.Bold(true).Render(last)
	// } else {
	// 	pkg = styles.Caller.Pkg.Render(last)
	// }

	// [icon] [package]

	var eproj string
	if e.enhancedProject == currentMainGoPackage {
		eproj = " "
	} else {
		eproj = " " + render(styles.Caller.Project, filepath.Base(e.enhancedProject)) + " "
	}

	return hyperlink("cursor://file/"+e.rawFilePath+":"+fmt.Sprintf("%d", e.rawFileLine), fmt.Sprintf("%s%s%s%s%s%s", projIcon, eproj, pkg, filePath, sep, num))
}

func NewEnhancedSource(pc uintptr) *EnhancedSource {
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()

	fullpkg, pkg, function := GetPackageAndFuncFromFuncName(frame.Function)

	return &EnhancedSource{
		ptr:             pc,
		rawFunc:         frame.Function,
		rawFilePath:     frame.File,
		rawFileLine:     frame.Line,
		enhancedFunc:    function,
		enhancedPkg:     pkg,
		enhancedProject: GetProjectFromPackage(fullpkg),
		enhancedFullPkg: fullpkg,
	}
}

func GetProjectFromPackage(pkg string) string {
	slashes := 3
	for _, project := range knownNestedProjects {
		if strings.HasPrefix(pkg, project) {
			slashes = strings.Count(pkg, "/") + 1
			break
		}
	}

	// if at least 3 slashes, return the first 2
	splt := strings.Split(pkg, "/")
	if len(splt) >= slashes {
		return strings.Join(splt[:slashes], "/")
	}

	return pkg
}

func GetPackageAndFuncFromFuncName(pc string) (fullpkg, pkg, function string) {
	funcName := pc
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}

	firstDot := strings.IndexByte(funcName[lastSlash:], '.') + lastSlash

	pkg = funcName[:firstDot]
	fname := funcName[firstDot+1:]

	if strings.Contains(pkg, ".(") {
		splt := strings.Split(pkg, ".(")
		pkg = splt[0]
		fname = "(" + splt[1] + "." + fname
	}

	fullpkg = pkg
	pkg = strings.TrimPrefix(pkg, currentMainGoPackage+"/")

	return fullpkg, pkg, fname
}

func FileNameOfPath(path string) string {
	tot := strings.Split(path, "/")
	if len(tot) > 1 {
		return tot[len(tot)-1]
	}

	return path
}
