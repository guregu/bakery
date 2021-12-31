package bakery

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path"
	"strings"

	"github.com/biscuit-auth/biscuit-go"
	"github.com/biscuit-auth/biscuit-go/parser"
)

// Bakery is a biscuit factory.
// Datalog files (recipes) are grouped together by folder (cookbooks).
type Bakery struct {
	recipes map[string]Cookbook
}

// New loads a new Bakery from the given file system and root directory.
// It will recursively walk through folders, looking for datalog files (recipes).
// Datalog files will be grouped by their parent folder, which becomes the name of the cookbook.
func New(fsys fs.FS, root string) (*Bakery, error) {
	b := &Bakery{
		recipes: make(map[string]Cookbook),
	}
	err := fs.WalkDir(fsys, root, func(filepath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		basedir := path.Base(path.Dir(filepath))
		f, err := fsys.Open(filepath)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		name := info.Name()
		name = name[:len(name)-len(path.Ext(name))]
		recipe, err := NewRecipe(name, f)
		if err != nil {
			return err
		}
		b.recipes[basedir] = append(b.recipes[basedir], recipe)
		log.Println("Bakery: loaded", basedir, name)
		return nil
	})
	return b, err
}

// Cookbook looks up a cookbook by name, corresponding to the directory name.
func (b *Bakery) Cookbook(name string) Cookbook {
	return b.recipes[name]
}

// Cookbook is a list of recipes.
type Cookbook []*Recipe

// Merged returns a recipe that is the result of merging all of this cookbook's recipes.
func (cb Cookbook) Merged() *Recipe {
	var result Recipe
	for _, r := range cb {
		result.merge(r)
	}
	return &result
}

// Find looks up a recipe with the given name. Recipe names have their file extension removed,
// so "foo.datalog" is found by Find("foo").
func (cb Cookbook) Find(name string) *Recipe {
	for _, r := range cb {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// NewRecipe loads a biscuit recipe with the given name from r.
// Recipes are Datalog files. Multiple Datalog statements may be separated via newline terminated semicolons.
func NewRecipe(name string, r io.Reader) (*Recipe, error) {
	scan := bufio.NewScanner(r)
	recipe := &Recipe{
		Name: name,
	}
	var text strings.Builder
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if strings.HasPrefix(line, "//") {
			// ignore comments
			continue
		}
		if strings.HasSuffix(line, ";") {
			line = strings.TrimSuffix(line, ";")
			text.WriteString(line)
			text.WriteRune('\n')
			if err := recipe.load(text.String()); err != nil {
				return nil, err
			}
			text.Reset()
			continue
		}
		text.WriteString(line)
		text.WriteRune('\n')
	}
	return recipe, scan.Err()
}

// Recipe is a collection of datalog objects.
// Think of it like a biscuit template, a cookie sheet if you will.
type Recipe struct {
	Name    string
	Facts   []biscuit.Fact
	Rules   []biscuit.Rule
	Caveats []biscuit.Caveat
}

// Apply adds all of this recipe's facts, rules, and caveats to the given verifier.
func (r *Recipe) Apply(b biscuit.Verifier) {
	for _, f := range r.Facts {
		b.AddFact(f)
	}
	for _, r := range r.Rules {
		b.AddRule(r)
	}
	for _, c := range r.Caveats {
		b.AddCaveat(c)
	}
}

// Build adds all of this recipe's facts, rules, and caveats as authority data to the given builder.
func (r *Recipe) Build(b biscuit.Builder) error {
	// TODO: AddAuthorityX will always shove #authority into the first term position, even if you add a rule like foo(#ambient, "bar")
	// ... which is not really a big issue in practice, but makes writing tests tricky
	for _, f := range r.Facts {
		if err := b.AddAuthorityFact(f); err != nil {
			return err
		}
	}
	for _, r := range r.Rules {
		if err := b.AddAuthorityRule(r); err != nil {
			return err
		}
	}
	for _, c := range r.Caveats {
		if err := b.AddAuthorityCaveat(c); err != nil {
			return err
		}
	}
	return nil
}

// BuildBlock adds all of this recipe's facts, rules, and caveats to the given block builder.
func (r *Recipe) BuildBlock(b biscuit.BlockBuilder) error {
	for _, f := range r.Facts {
		if err := b.AddFact(f); err != nil {
			return err
		}
	}
	for _, r := range r.Rules {
		if err := b.AddRule(r); err != nil {
			return err
		}
	}
	for _, c := range r.Caveats {
		if err := b.AddCaveat(c); err != nil {
			return err
		}
	}
	return nil
}

func (b *Recipe) merge(other *Recipe) {
	b.Facts = append(b.Facts, other.Facts...)
	b.Rules = append(b.Rules, other.Rules...)
	b.Caveats = append(b.Caveats, other.Caveats...)
}

func (b *Recipe) load(text string) error {
	text = strings.Trim(text, "\n")
	p := parser.New()

	switch {
	// caveat
	case strings.HasPrefix(text, "["):
		c, err := p.Caveat(text)
		if err != nil {
			return fmt.Errorf("load caveat: %w", err)
		}
		b.Caveats = append(b.Caveats, c)

	// rule
	case strings.Contains(text, "<-"):
		r, err := p.Rule(text)
		if err != nil {
			return fmt.Errorf("load rule: %w", err)
		}
		b.Rules = append(b.Rules, r)

	// fact
	default:
		f, err := p.Fact(text)
		if err != nil {
			return fmt.Errorf("load fact: %w", err)
		}
		b.Facts = append(b.Facts, f)
	}

	return nil
}
