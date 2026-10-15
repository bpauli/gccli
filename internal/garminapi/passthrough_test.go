package garminapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var passthroughMethods = []string{
	"AddEvent", "DeleteBloodPressure", "DeleteEvent", "DeleteWeight", "DeleteWorkout", "GetBloodPressure",
	"GetBodyComposition", "GetDailyWeighIns", "GetDeviceSettings", "GetDeviceSolar", "GetDevices", "GetEvents",
	"GetGear", "GetGearActivities", "GetGearDefaults", "GetGearStats", "GetHydration", "GetLastUsedDevice",
	"GetNutritionDailyFoodLog", "GetNutritionDailyMeals", "GetNutritionDailySettings",
	"GetPrimaryTrainingDevice", "GetProfile", "GetUserSettings", "GetWeighIns", "GetWorkout", "GetWorkouts",
	"LinkGear", "ScheduleWorkout", "UnlinkGear", "UnscheduleWorkout", "UpdateWorkout", "UploadWorkout",
}

func TestNoNewPassthroughMethods(t *testing.T) {
	fset := token.NewFileSet()
	got := map[string]token.Position{}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && isPassthrough(fn) {
				got[fn.Name.Name] = fset.Position(fn.Pos())
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(got)) {
		if !slices.Contains(passthroughMethods, name) {
			t.Errorf("%s: Client.%s only forwards one request: add a row to the service's endpoint table and call it with Client.Call", got[name], name)
		}
	}
	for _, name := range passthroughMethods {
		if _, ok := got[name]; !ok {
			t.Errorf("passthroughMethods lists %s, which is no longer a passthrough method: delete it from the list", name)
		}
	}
}

const maxPassthroughStatements = 3

func isPassthrough(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || !fn.Name.IsExported() || fn.Body == nil || len(fn.Body.List) > maxPassthroughStatements {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok || len(fn.Recv.List[0].Names) == 0 {
		return false
	}
	if id, ok := star.X.(*ast.Ident); !ok || id.Name != "Client" {
		return false
	}
	recv := fn.Recv.List[0].Names[0].Name
	var onRecv []string
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == recv {
					onRecv = append(onRecv, sel.Sel.Name)
				}
			}
		}
		return true
	})
	return len(onRecv) == 1 && (onRecv[0] == "ConnectAPI" || onRecv[0] == "Call")
}
