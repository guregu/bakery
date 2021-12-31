## bakery [![GoDoc](https://godoc.org/github.com/guregu/bakery?status.svg)](https://godoc.org/github.com/guregu/bakery)
`import "github.com/guregu/bakery"`

bakery is a small helper library to assist with wrangling [Biscuits](https://github.com/biscuit-auth/biscuit-go). Very experimental, expect this to change as I learn more about Biscuits.

This library lets you organize your Datalog facts, rules, and caveats in a file system. A **recipe** is a file, containing multiple Datalog statements separated by semicolons. A **cookbook** is a folder containing multiple recipes. The **bakery** keeps track of your cookbooks.

### Example
```go
import (
	"embed"

	"github.com/biscuit-auth/biscuit-go"
	"github.com/biscuit-auth/biscuit-go/sig"
	"github.com/guregu/bakery"
)

//go:embed bakery
var bakeryFS embed.FS

func main() {
	// a bakery is a collection of cookbooks
	bake, err := bakery.New(bakeryFS, "bakery")
	if err != nil {
		t.Fatal(err)
	}
	
	// a cookbook is a collection of datalog recipes
	cookbook := bake.Cookbook("hierarchy")
	// a recipe is a bundle of datalog data that can be applied to new biscuits, verifiers, etc
	recipe := cookbook.Find("base")

	// create a new biscuit via the "base" recipe
	keys := sig.GenerateKeypair(nil)
	builder := biscuit.NewBuilder(keys)
	recipe.Build(builder) // apply recipe to biscuit builder
	bisc, _ := builder.Build()
}
```

