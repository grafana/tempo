package fileprocessor

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/manuelarte/funcorder/internal/astutils"
	"github.com/manuelarte/funcorder/internal/features"
	"github.com/manuelarte/funcorder/internal/models"
)

// FileProcessor Holder to store all the functions that are potential to be constructors and all the structs.
type FileProcessor struct {
	structs  map[string]*models.StructHolder
	features features.Feature
}

// NewFileProcessor creates a new file processor.
func NewFileProcessor(checkers features.Feature) *FileProcessor {
	return &FileProcessor{
		structs:  make(map[string]*models.StructHolder),
		features: checkers,
	}
}

// Analyze check whether the order of the methods in the constructor is correct.
func (fp *FileProcessor) Analyze() []analysis.Diagnostic {
	var reports []analysis.Diagnostic

	for _, sh := range fp.structs {
		// filter out structs that are not declared inside that file
		if sh.Struct != nil {
			reports = append(reports, sh.Analyze()...)
		}
	}

	return reports
}

func (fp *FileProcessor) addConstructor(sc models.StructConstructor) {
	sh := fp.getOrCreate(sc.GetStructReturn().Name)
	sh.AddConstructor(sc.GetConstructor())
}

func (fp *FileProcessor) addMethod(st string, n *ast.FuncDecl) {
	sh := fp.getOrCreate(st)
	sh.AddMethod(n)
}

func (fp *FileProcessor) NewFileNode(_ *ast.File) {
	fp.structs = make(map[string]*models.StructHolder)
}

func (fp *FileProcessor) NewFuncDecl(n *ast.FuncDecl) {
	if sc, ok := models.NewStructConstructor(n); ok {
		fp.addConstructor(sc)
		return
	}

	if st, ok := astutils.FuncIsMethod(n); ok {
		fp.addMethod(st.Name, n)
	}
}

func (fp *FileProcessor) NewTypeSpec(n *ast.TypeSpec) {
	sh := fp.getOrCreate(n.Name.Name)
	sh.Struct = n
}

func (fp *FileProcessor) getOrCreate(structName string) *models.StructHolder {
	if holder, ok := fp.structs[structName]; ok {
		return holder
	}

	created := &models.StructHolder{Features: fp.features}
	fp.structs[structName] = created

	return created
}
