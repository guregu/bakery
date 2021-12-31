package bakery

import (
	"embed"
	"strings"
	"testing"

	"github.com/biscuit-auth/biscuit-go"
	"github.com/biscuit-auth/biscuit-go/sig"
	"github.com/davecgh/go-spew/spew"
)

//go:embed testdata
var testdata embed.FS

func TestBakery(t *testing.T) {
	bake, err := New(testdata, "testdata")
	if err != nil {
		t.Fatal(err)
	}
	spew.Dump(bake)

	recipes := bake.Cookbook("hierarchy")
	recipe := recipes.Find("base")

	keys := sig.GenerateKeypair(nil)
	build := biscuit.NewBuilder(keys)
	recipe.Build(build)

	bisc, err := build.Build()
	if err != nil {
		t.Fatal(err)
	}

	v, err := bisc.Verify(keys.Public())
	if err != nil {
		t.Fatal(err)
	}

	// add ambient data to verifier
	recipes.Find("ambient").Apply(v)

	// run query: what secrets can we view?
	factset, err := v.Query(recipes.Find("rights").Rules[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Query results: %+v", factset)

	if err := v.Verify(); err != nil {
		t.Error(err)
	}
}

func TestLoadRecipe(t *testing.T) {
	const testData = `
	// comment
	human(#authority, "socrates");

	valid(#authority) <- 
		realm(#ambient, $realm),
		realm(#authority, $realm),
		client(#ambient, $client),
		client(#authority, $client);

	[right(#authority, $file, #read) <- 
		resource(#ambient, $file), 
		owner(#ambient, $user, $file) 
		@ $user == "username", prefix($file, "/home/username")];
	`

	r := strings.NewReader(testData)

	b, err := NewRecipe("test", r)
	if err != nil {
		t.Error(err)
	}

	spew.Dump(b)
}
