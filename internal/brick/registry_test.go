package brick_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
)

// constructedSampleTestDouble records the parameters its constructor received.
type constructedSampleTestDouble struct {
	received sampleParameters
}

func constructSampleTestDouble(parameters sampleParameters, _ brick.Ports) (any, error) {
	return &constructedSampleTestDouble{received: parameters}, nil
}

func TestRegister_DerivesSchemaAndConstructsWithTypedParameters(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := brick.Register(registry, "sample_kind", constructSampleTestDouble)
	if registerError != nil {
		t.Fatalf("registering: %v", registerError)
	}

	kind, kindFound := registry.Kind("sample_kind")
	if !kindFound {
		t.Fatal("want the kind registered")
	}

	if _, addressFound := kind.Schema().Field(addressFieldName); !addressFound {
		t.Fatal("want the schema derived from the parameter struct")
	}

	parameters := kind.Schema().NewParameters()

	addressField, _ := kind.Schema().Field(addressFieldName)

	assignError := kind.Schema().Assign(parameters, addressField, sampleAddress)
	if assignError != nil {
		t.Fatalf("assigning: %v", assignError)
	}

	noPorts := brick.Ports{URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil}

	constructed, constructError := kind.Construct(parameters, noPorts)
	if constructError != nil {
		t.Fatalf("constructing: %v", constructError)
	}

	testDouble, isTestDouble := constructed.(*constructedSampleTestDouble)
	if !isTestDouble || testDouble.received.Address != sampleAddress {
		t.Fatalf("want the constructor to receive the typed parameters, got %+v", constructed)
	}
}

func TestRegister_RejectsDuplicateKind(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	firstError := brick.Register(registry, "sample_kind", constructSampleTestDouble)
	if firstError != nil {
		t.Fatalf("first registration: %v", firstError)
	}

	secondError := brick.Register(registry, "sample_kind", constructSampleTestDouble)
	if !errors.Is(secondError, brick.ErrInvalidKind) {
		t.Fatalf("want ErrInvalidKind for a duplicate, got %v", secondError)
	}
}

func TestRegister_RejectsBadKindName(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	for _, badName := range []string{"", "Sample", "sample kind", "9sample"} {
		registerError := brick.Register(registry, badName, constructSampleTestDouble)
		if !errors.Is(registerError, brick.ErrInvalidKind) {
			t.Errorf("want ErrInvalidKind for %q, got %v", badName, registerError)
		}
	}
}

func TestRegister_RejectsInvalidParameterDeclaration(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	constructBroken := func(_ missingDescription, _ brick.Ports) (any, error) { return struct{}{}, nil }

	registerError := brick.Register(registry, "broken_kind", constructBroken)
	if !errors.Is(registerError, brick.ErrInvalidParameterDeclaration) {
		t.Fatalf("want ErrInvalidParameterDeclaration, got %v", registerError)
	}

	if _, kindFound := registry.Kind("broken_kind"); kindFound {
		t.Fatal("want a rejected kind left unregistered")
	}
}

func TestRegistry_KindNamesAreSorted(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	for _, kindName := range []string{"zeta_kind", "alpha_kind", "mid_kind"} {
		registerError := brick.Register(registry, kindName, constructSampleTestDouble)
		if registerError != nil {
			t.Fatalf("registering %s: %v", kindName, registerError)
		}
	}

	want := []string{"alpha_kind", "mid_kind", "zeta_kind"}
	if got := registry.KindNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestKindConstruct_RejectsParametersOfTheWrongType(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerError := brick.Register(registry, "sample_kind", constructSampleTestDouble)
	if registerError != nil {
		t.Fatalf("registering: %v", registerError)
	}

	kind, _ := registry.Kind("sample_kind")

	noPorts := brick.Ports{URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil}

	_, constructError := kind.Construct(&missingDescription{Address: ""}, noPorts)
	if constructError == nil {
		t.Fatal("want an error for parameters of another kind")
	}
}

// TestRegisterScoped_KeepsAnAppsBricksToItself: an application's own bricks,
// such as GoLand's actions, are kinds like any other but carry their app set's
// scope, so the editor offers them only on that application's screen
// testAppSetID is the app set the scope tests register.
const testAppSetID = "goland"

// (evaluation, app screens).
func TestRegisterScoped_KeepsAnAppsBricksToItself(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	registerErrors := []error{
		brick.Register(registry, "general_kind", constructSampleTestDouble),
		brick.RegisterScoped(registry, testAppSetID, "goland_build", constructSampleTestDouble),
	}
	for _, registerError := range registerErrors {
		if registerError != nil {
			t.Fatal(registerError)
		}
	}

	if got := registry.KindNamesIn(""); !reflect.DeepEqual(got, []string{"general_kind"}) {
		t.Errorf("want only the general kind listed generally, got %v", got)
	}

	if got := registry.KindNamesIn(testAppSetID); !reflect.DeepEqual(got, []string{"goland_build"}) {
		t.Errorf("want GoLand's kind listed under its scope, got %v", got)
	}

	kind, _ := registry.Kind("goland_build")
	if kind.Scope() != testAppSetID {
		t.Errorf("want the kind to carry its scope, got %q", kind.Scope())
	}

	if len(registry.KindNames()) != 2 {
		t.Error("want every kind still known, for decoding and the reference")
	}
}

// TestAppSets_MatchTheirApplications: an app set names the applications it is
// for, as a key's application field holds them: a bundle identifier on macOS,
// an executable's path on Windows. One set may serve several applications,
// each named for itself.
func TestAppSets_MatchTheirApplications(t *testing.T) {
	t.Parallel()

	registry := brick.NewRegistry()

	setError := registry.RegisterAppSet(brick.AppSet{
		ID: testAppSetID, Name: "JetBrains IDEs", Groups: nil,
		Applications: []brick.AppIdentity{
			{Name: "GoLand", Identifiers: []string{"com.jetbrains.goland", "goland64.exe"}},
			{Name: "Rider", Identifiers: []string{"com.jetbrains.rider"}},
		},
	})
	if setError != nil {
		t.Fatal(setError)
	}

	for identifier, want := range map[string]bool{
		"com.jetbrains.goland":                               true,
		"COM.JETBRAINS.GOLAND":                               true,
		`C:\Program Files\JetBrains\GoLand\bin\goland64.exe`: true,
		"com.jetbrains.rider":                                true,
		"com.jetbrains.pycharm":                              false,
		"":                                                   false,
	} {
		set, found := registry.AppSetFor(identifier)
		if found != want || (found && set.ID != testAppSetID) {
			t.Errorf("%q: want matched %v, got %v (%+v)", identifier, want, found, set)
		}
	}

	set, _ := registry.AppSetFor("com.jetbrains.rider")
	for identifier, wantName := range map[string]string{
		"COM.JETBRAINS.RIDER":        "Rider",
		`D:\GoLand\bin\goland64.exe`: "GoLand",
		"com.example.other":          "JetBrains IDEs",
	} {
		if name := set.NameFor(identifier); name != wantName {
			t.Errorf("%q: want named %q, got %q", identifier, wantName, name)
		}
	}

	if registry.RegisterAppSet(brick.AppSet{ID: testAppSetID, Name: "Again", Applications: nil, Groups: nil}) == nil {
		t.Error("want a second set with the same ID refused")
	}
}
