package main

import (
	"bufio"
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/nikolaydubina/go-binsize-treemap/fmtbytecount"
	"github.com/nikolaydubina/go-binsize-treemap/symtab"
	"github.com/nikolaydubina/treemap"
	"github.com/nikolaydubina/treemap/render"
)

const doc string = `
Go binary size treemap.

Examples
$ go tool nm -size <binary finename> | go-binsize-treemap > binsize.svg
$ go tool nm -size <binary finename> | c++filt | go-binsize-treemap > binsize.svg

Command options:
`

var grey = color.RGBA{128, 128, 128, 255}

func main() {
	var (
		coverprofile       string
		w                  float64
		h                  float64
		marginBox          float64
		paddingBox         float64
		padding            float64
		maxDepth           uint
		includeSymbols     bool
		includeUnknown     bool
		includePureSymbols bool
		outputCSV          bool
		verbosity          uint
	)

	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), doc)
		flag.PrintDefaults()
	}
	flag.StringVar(&coverprofile, "coverprofile", "", "filename of input coverprofile (e.g. cover.out)")
	flag.Float64Var(&w, "w", 1028, "width of output")
	flag.Float64Var(&h, "h", 640, "height of output")
	flag.Float64Var(&marginBox, "margin-box", 4, "margin between boxes")
	flag.Float64Var(&paddingBox, "padding-box", 4, "padding between box border and content")
	flag.Float64Var(&padding, "padding", 16, "padding around root content")
	flag.UintVar(&maxDepth, "max-depth", 0, "if zero then no max depth is set, else will show only number of levels from root including")
	flag.BoolVar(&includeSymbols, "symbols", false, "include leaf symbols or not")
	flag.BoolVar(&includePureSymbols, "symbols-pure", false, "include symbols that do not have package, likely non Go symbols")
	flag.BoolVar(&outputCSV, "csv", false, "print as csv instead")
	flag.UintVar(&verbosity, "v", 0, "verbosity level of logging, the higher the more logs")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	parser := symtab.GoSymtabParser{
		Verbosity: verbosity,
	}
	symtabFile, err := parser.ParseSymtab(lines)
	if err != nil || symtabFile == nil {
		log.Fatal(err)
	}

	converter := symtab.BasicSymtabConverter{
		MaxDepth:           maxDepth,
		IncludeSymbols:     includeSymbols,
		IncludeUnknown:     includeUnknown,
		IncludePureSymbols: includePureSymbols,
		Verbosity:          verbosity,
	}
	tree := converter.SymtabFileToTreemap(*symtabFile)

	sizeImputer := treemap.SumSizeImputer{EmptyLeafSize: 0}
	sizeImputer.ImputeSize(tree)

	updateNodeNamesWithByteSize(&tree)
	treemap.CollapseRoot(&tree)

	if outputCSV {
		for name, node := range tree.Nodes {
			fmt.Printf("%s,%f\n", name, node.Size)
		}
		return
	}

	uiBuilder := render.UITreeMapBuilder{
		Colorer:     render.NoneColorer{},
		BorderColor: grey,
	}
	spec := uiBuilder.NewUITreeMap(tree, w, h, marginBox, paddingBox, padding)
	renderer := render.SVGRenderer{}

	os.Stdout.Write(renderer.Render(spec, w, h))
}

func updateNodeNamesWithByteSize(tree *treemap.Tree) {
	for name, node := range tree.Nodes {
		count, suffix := fmtbytecount.ByteCountIEC(uint(math.Floor(node.Size)))
		nameWithSize := fmt.Sprintf("%s %.2f%sB", node.Name, count, suffix)

		// for secret root just size
		if name == symtab.RootNodeName {
			nameWithSize = fmt.Sprintf("%.2f%sB", count, suffix)
		}

		tree.Nodes[name] = treemap.Node{
			Path:    node.Path,
			Name:    nameWithSize,
			Size:    node.Size,
			Heat:    node.Heat,
			HasHeat: node.HasHeat,
		}
	}
}
