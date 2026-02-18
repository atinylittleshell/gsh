package interpreter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestImportExportBasic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gsh-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	helperContent := `
export helperVar = 42

export tool helperFunc(x) {
    return x * 2
}

privateVar = "not exported"
`
	helperPath := filepath.Join(tmpDir, "helper.gsh")
	if err := os.WriteFile(helperPath, []byte(helperContent), 0644); err != nil {
		t.Fatalf("Failed to write helper.gsh: %v", err)
	}

	mainContent := `
import { helperVar, helperFunc } from "./helper.gsh"
result = helperFunc(helperVar)
`

	interp := New(nil)
	defer interp.Close()

	result, err := interp.EvalString(mainContent, &ScriptOrigin{
		Type:     OriginFilesystem,
		BasePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	vars := result.Variables()
	if resultVal, ok := vars["result"]; ok {
		if numVal, ok := resultVal.(*NumberValue); ok {
			if numVal.Value != 84 {
				t.Errorf("Expected result to be 84, got %v", numVal.Value)
			}
		} else {
			t.Errorf("Expected result to be a number, got %T", resultVal)
		}
	} else {
		t.Errorf("Expected 'result' variable to be defined")
	}
}

func TestImportSideEffectOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gsh-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	moduleContent := `sideEffectVar = "executed"`
	modulePath := filepath.Join(tmpDir, "sideeffect.gsh")
	if err := os.WriteFile(modulePath, []byte(moduleContent), 0644); err != nil {
		t.Fatalf("Failed to write module: %v", err)
	}

	mainContent := `import "./sideeffect.gsh"`

	interp := New(nil)
	defer interp.Close()

	_, err = interp.EvalString(mainContent, &ScriptOrigin{
		Type:     OriginFilesystem,
		BasePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	if _, ok := interp.globalEnv.Get("sideEffectVar"); ok {
		t.Errorf("Side effect import should not bring variables into scope")
	}
}

func TestImportNonExportedSymbol(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gsh-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	helperContent := `privateVar = "not exported"`
	helperPath := filepath.Join(tmpDir, "helper.gsh")
	if err := os.WriteFile(helperPath, []byte(helperContent), 0644); err != nil {
		t.Fatalf("Failed to write helper.gsh: %v", err)
	}

	mainContent := `import { privateVar } from "./helper.gsh"`

	interp := New(nil)
	defer interp.Close()

	_, err = interp.EvalString(mainContent, &ScriptOrigin{
		Type:     OriginFilesystem,
		BasePath: tmpDir,
	})
	if err == nil {
		t.Errorf("Expected error when importing non-exported symbol")
	}
}

func TestCircularImportDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gsh-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	aContent := `
import "./b.gsh"
export aVar = 1
`
	aPath := filepath.Join(tmpDir, "a.gsh")
	if err := os.WriteFile(aPath, []byte(aContent), 0644); err != nil {
		t.Fatalf("Failed to write a.gsh: %v", err)
	}

	bContent := `
import "./a.gsh"
export bVar = 2
`
	bPath := filepath.Join(tmpDir, "b.gsh")
	if err := os.WriteFile(bPath, []byte(bContent), 0644); err != nil {
		t.Fatalf("Failed to write b.gsh: %v", err)
	}

	content, err := os.ReadFile(aPath)
	if err != nil {
		t.Fatalf("Failed to read a.gsh: %v", err)
	}

	interp := New(nil)
	defer interp.Close()

	_, err = interp.EvalString(string(content), &ScriptOrigin{
		Type:     OriginFilesystem,
		BasePath: tmpDir,
	})
	if err == nil {
		t.Errorf("Expected circular import error")
	}
}

func TestExportTool(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gsh-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	helperContent := `
export tool add(a, b) {
    return a + b
}
`
	helperPath := filepath.Join(tmpDir, "helper.gsh")
	if err := os.WriteFile(helperPath, []byte(helperContent), 0644); err != nil {
		t.Fatalf("Failed to write helper.gsh: %v", err)
	}

	mainContent := `
import { add } from "./helper.gsh"
result = add(10, 20)
`

	interp := New(nil)
	defer interp.Close()

	result, err := interp.EvalString(mainContent, &ScriptOrigin{
		Type:     OriginFilesystem,
		BasePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	vars := result.Variables()
	if resultVal, ok := vars["result"]; ok {
		if numVal, ok := resultVal.(*NumberValue); ok {
			if numVal.Value != 30 {
				t.Errorf("Expected result to be 30, got %v", numVal.Value)
			}
		} else {
			t.Errorf("Expected result to be a number, got %T", resultVal)
		}
	} else {
		t.Errorf("Expected 'result' variable to be defined")
	}
}

func TestEmbeddedImport(t *testing.T) {
	embedFS := fstest.MapFS{
		"defaults/init.gsh": &fstest.MapFile{
			Data: []byte(`
import { helperFunc } from "./helpers/utils.gsh"
result = helperFunc(10)
`),
		},
		"defaults/helpers/utils.gsh": &fstest.MapFile{
			Data: []byte(`
export tool helperFunc(x) {
    return x * 3
}
`),
		},
	}

	initContent, _ := embedFS.ReadFile("defaults/init.gsh")

	interp := New(nil)
	defer interp.Close()

	result, err := interp.EvalString(string(initContent), &ScriptOrigin{
		Type:     OriginEmbed,
		BasePath: "defaults",
		EmbedFS:  embedFS,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	vars := result.Variables()
	if resultVal, ok := vars["result"]; ok {
		if numVal, ok := resultVal.(*NumberValue); ok {
			if numVal.Value != 30 {
				t.Errorf("Expected result to be 30, got %v", numVal.Value)
			}
		} else {
			t.Errorf("Expected result to be a number, got %T", resultVal)
		}
	} else {
		t.Errorf("Expected 'result' variable to be defined")
	}
}

func TestEmbeddedImportNestedRelativePath(t *testing.T) {
	embedFS := fstest.MapFS{
		"defaults/events/agent.gsh": &fstest.MapFile{
			Data: []byte(`
import { sharedValue } from "../shared.gsh"
result = sharedValue + 100
`),
		},
		"defaults/shared.gsh": &fstest.MapFile{
			Data: []byte(`export sharedValue = 42`),
		},
	}

	agentContent, _ := embedFS.ReadFile("defaults/events/agent.gsh")

	interp := New(nil)
	defer interp.Close()

	result, err := interp.EvalString(string(agentContent), &ScriptOrigin{
		Type:     OriginEmbed,
		BasePath: "defaults/events",
		EmbedFS:  embedFS,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	vars := result.Variables()
	if resultVal, ok := vars["result"]; ok {
		if numVal, ok := resultVal.(*NumberValue); ok {
			if numVal.Value != 142 {
				t.Errorf("Expected result to be 142, got %v", numVal.Value)
			}
		} else {
			t.Errorf("Expected result to be a number, got %T", resultVal)
		}
	} else {
		t.Errorf("Expected 'result' variable to be defined")
	}
}

func TestEmbeddedImportNotConfigured(t *testing.T) {
	interp := New(nil)
	defer interp.Close()

	_, err := interp.EvalString(`import "./helper.gsh"`, &ScriptOrigin{
		Type:     OriginEmbed,
		BasePath: "defaults",
		// EmbedFS intentionally not set
	})
	if err == nil {
		t.Errorf("Expected error when embedFS not configured")
	}
	if !strings.Contains(err.Error(), "embedded filesystem not configured") {
		t.Errorf("Expected 'embedded filesystem not configured' error, got: %v", err)
	}
}

func TestRecursiveImportScoping(t *testing.T) {
	// Test that A imports B imports C, and:
	// - C's exports are available in B's scope
	// - B's exports (which may use C's exports) are available in A's scope
	// - C's exports are NOT directly visible in A's scope
	tmpDir, err := os.MkdirTemp("", "gsh-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Module C: exports cValue
	cContent := `export cValue = 100`
	cPath := filepath.Join(tmpDir, "c.gsh")
	if err := os.WriteFile(cPath, []byte(cContent), 0644); err != nil {
		t.Fatalf("Failed to write c.gsh: %v", err)
	}

	// Module B: imports cValue from C, exports bValue (which uses cValue)
	bContent := `
import { cValue } from "./c.gsh"
export bValue = cValue + 50
`
	bPath := filepath.Join(tmpDir, "b.gsh")
	if err := os.WriteFile(bPath, []byte(bContent), 0644); err != nil {
		t.Fatalf("Failed to write b.gsh: %v", err)
	}

	// Module A (main): imports bValue from B
	aContent := `
import { bValue } from "./b.gsh"
result = bValue
`

	interp := New(nil)
	defer interp.Close()

	result, err := interp.EvalString(aContent, &ScriptOrigin{
		Type:     OriginFilesystem,
		BasePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	vars := result.Variables()

	// Check that bValue is accessible and has correct value (100 + 50 = 150)
	if resultVal, ok := vars["result"]; ok {
		if numVal, ok := resultVal.(*NumberValue); ok {
			if numVal.Value != 150 {
				t.Errorf("Expected result to be 150, got %v", numVal.Value)
			}
		} else {
			t.Errorf("Expected result to be a number, got %T", resultVal)
		}
	} else {
		t.Errorf("Expected 'result' variable to be defined")
	}

	// Check that cValue is NOT directly visible in A's scope
	if _, ok := vars["cValue"]; ok {
		t.Errorf("cValue should NOT be visible in A's scope (it was not imported)")
	}

	// Also verify via interpreter's env that cValue is not accessible
	if _, ok := interp.globalEnv.Get("cValue"); ok {
		t.Errorf("cValue should NOT be in the interpreter's environment after A's execution")
	}
}

func TestRecursiveImportReExport(t *testing.T) {
	// Test that B can re-export a value from C, making it available in A
	tmpDir, err := os.MkdirTemp("", "gsh-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Module C: exports cFunc
	cContent := `
export tool cFunc(x) {
    return x * 2
}
`
	cPath := filepath.Join(tmpDir, "c.gsh")
	if err := os.WriteFile(cPath, []byte(cContent), 0644); err != nil {
		t.Fatalf("Failed to write c.gsh: %v", err)
	}

	// Module B: imports cFunc from C, uses it in bFunc, and exports bFunc
	bContent := `
import { cFunc } from "./c.gsh"
export tool bFunc(x) {
    return cFunc(x) + 10
}
`
	bPath := filepath.Join(tmpDir, "b.gsh")
	if err := os.WriteFile(bPath, []byte(bContent), 0644); err != nil {
		t.Fatalf("Failed to write b.gsh: %v", err)
	}

	// Module A (main): imports bFunc from B and uses it
	aContent := `
import { bFunc } from "./b.gsh"
result = bFunc(5)
`

	interp := New(nil)
	defer interp.Close()

	result, err := interp.EvalString(aContent, &ScriptOrigin{
		Type:     OriginFilesystem,
		BasePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	vars := result.Variables()

	// Check that bFunc works correctly: cFunc(5) = 10, bFunc(5) = 10 + 10 = 20
	if resultVal, ok := vars["result"]; ok {
		if numVal, ok := resultVal.(*NumberValue); ok {
			if numVal.Value != 20 {
				t.Errorf("Expected result to be 20, got %v", numVal.Value)
			}
		} else {
			t.Errorf("Expected result to be a number, got %T", resultVal)
		}
	} else {
		t.Errorf("Expected 'result' variable to be defined")
	}

	// cFunc should NOT be visible in A's scope
	if _, ok := vars["cFunc"]; ok {
		t.Errorf("cFunc should NOT be visible in A's scope")
	}
}

func TestRelativeImportPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gsh-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	helperContent := `export parentVar = "from parent"`
	helperPath := filepath.Join(tmpDir, "helper.gsh")
	if err := os.WriteFile(helperPath, []byte(helperContent), 0644); err != nil {
		t.Fatalf("Failed to write helper.gsh: %v", err)
	}

	mainContent := `
import { parentVar } from "../helper.gsh"
result = parentVar
`

	interp := New(nil)
	defer interp.Close()

	result, err := interp.EvalString(mainContent, &ScriptOrigin{
		Type:     OriginFilesystem,
		BasePath: subDir,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	vars := result.Variables()
	if resultVal, ok := vars["result"]; ok {
		if strVal, ok := resultVal.(*StringValue); ok {
			if strVal.Value != "from parent" {
				t.Errorf("Expected result to be 'from parent', got %v", strVal.Value)
			}
		} else {
			t.Errorf("Expected result to be a string, got %T", resultVal)
		}
	} else {
		t.Errorf("Expected 'result' variable to be defined")
	}
}

func TestDefaultsStructureImport(t *testing.T) {
	// Test that simulates the new defaults/ structure with init.gsh importing modules
	embedFS := fstest.MapFS{
		"defaults/init.gsh": &fstest.MapFile{
			Data: []byte(`
# Entry point that imports modules
import { lite, workhorse } from "./models.gsh"
import "./events/agent.gsh"

# Store models in variables (not SDK properties which require REPL context)
liteModel = lite
workhorseModel = workhorse
`),
		},
		"defaults/models.gsh": &fstest.MapFile{
			Data: []byte(`
export model lite {
    provider: "openai",
    apiKey: "test",
    model: "test-lite",
}

export model workhorse {
    provider: "openai",
    apiKey: "test",
    model: "test-workhorse",
}
`),
		},
		"defaults/events/agent.gsh": &fstest.MapFile{
			Data: []byte(`
# Side-effect import - registers event handler
tool onAgentStart(ctx, next) {
    # Handler registered
    return next(ctx)
}
gsh.use("agent.start", onAgentStart)
`),
		},
	}

	initContent, _ := embedFS.ReadFile("defaults/init.gsh")

	interp := New(nil)
	defer interp.Close()

	_, err := interp.EvalString(string(initContent), &ScriptOrigin{
		Type:     OriginEmbed,
		BasePath: "defaults",
		EmbedFS:  embedFS,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	// Verify models were imported and assigned
	vars := interp.GetVariables()
	if _, ok := vars["liteModel"]; !ok {
		t.Errorf("Expected 'liteModel' to be available")
	}
	if _, ok := vars["workhorseModel"]; !ok {
		t.Errorf("Expected 'workhorseModel' to be available")
	}

	// Verify event handler was registered
	handlers := interp.GetEventHandlers("agent.start")
	if len(handlers) == 0 {
		t.Errorf("Expected agent.start event handler to be registered")
	}
}

func TestCrossModuleImportWithSubdirectories(t *testing.T) {
	// Test importing between subdirectories (e.g., middleware imports from models)
	embedFS := fstest.MapFS{
		"defaults/models.gsh": &fstest.MapFile{
			Data: []byte(`
export model testModel {
    provider: "openai",
    apiKey: "test",
    model: "test-model",
}
`),
		},
		"defaults/middleware.gsh": &fstest.MapFile{
			Data: []byte(`
import { testModel } from "./models.gsh"
export usedModel = testModel
`),
		},
	}

	middlewareContent, _ := embedFS.ReadFile("defaults/middleware.gsh")

	interp := New(nil)
	defer interp.Close()

	result, err := interp.EvalString(string(middlewareContent), &ScriptOrigin{
		Type:     OriginEmbed,
		BasePath: "defaults",
		EmbedFS:  embedFS,
	})
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	// Verify model was imported and exported
	vars := result.Variables()
	if _, ok := vars["usedModel"]; !ok {
		t.Errorf("Expected 'usedModel' to be exported")
	}
}
