package tool

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverSkills(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(filepath.Join(directory, "alpha"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "beta"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "alpha", "SKILL.md"), []byte(`---
name: alpha
description: Handle alpha tasks.
---

Alpha instructions.
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "beta", "SKILL.md"), []byte(`---
name: beta
description: Handle beta tasks.
metadata:
  ignored: true
---

Beta instructions.
`), 0o600); err != nil {
		t.Fatal(err)
	}

	skills, skillErrors := DiscoverSkills(directory)
	if len(skillErrors) != 0 {
		t.Fatalf("skill errors = %v", skillErrors)
	}
	want := []Skill{
		{
			Name:        "alpha",
			Description: "Handle alpha tasks.",
			Path:        filepath.Join(directory, "alpha", "SKILL.md"),
		},
		{
			Name:        "beta",
			Description: "Handle beta tasks.",
			Path:        filepath.Join(directory, "beta", "SKILL.md"),
		},
	}
	if got := skills; !reflect.DeepEqual(got, want) {
		t.Fatalf("skills = %#v, want %#v", got, want)
	}
}

func TestDiscoverSkillsAllowsMissingDirectory(t *testing.T) {
	skills, skillErrors := DiscoverSkills(filepath.Join(t.TempDir(), "missing"))
	if len(skillErrors) != 0 {
		t.Fatalf("skill errors = %v", skillErrors)
	}
	if got := skills; len(got) != 0 {
		t.Fatalf("skills = %#v, want empty", got)
	}
}

func TestDiscoverSkillsContinuesAfterInvalidFrontmatter(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "skills")
	invalidPath := filepath.Join(directory, "invalid", "SKILL.md")
	validPath := filepath.Join(directory, "valid", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(invalidPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(validPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidPath, []byte("name: invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validPath, []byte("---\nname: valid\ndescription: Valid skill.\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	skills, skillErrors := DiscoverSkills(directory)
	want := []Skill{{Name: "valid", Description: "Valid skill.", Path: validPath}}
	if got := skills; !reflect.DeepEqual(got, want) {
		t.Fatalf("skills = %#v, want %#v", got, want)
	}
	if len(skillErrors) != 1 ||
		!strings.Contains(skillErrors[0].Error(), "missing opening YAML frontmatter delimiter") {
		t.Fatalf("skill errors = %v", skillErrors)
	}
}

func TestDiscoverSkillsRejectsInvalidMetadataAndDuplicateNames(t *testing.T) {
	directory := t.TempDir()
	for _, entry := range []struct {
		name, contents string
	}{
		{"alpha", "---\nname: review\ndescription: Review code.\n---\n"},
		{"duplicate", "---\nname: review\ndescription: Duplicate.\n---\n"},
		{"missing-name", "---\ndescription: Missing name.\n---\n"},
		{"missing-description", "---\nname: missing-description\n---\n"},
	} {
		path := filepath.Join(directory, entry.name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(entry.contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	skills, skillErrors := DiscoverSkills(directory)
	want := []Skill{{Name: "review", Description: "Review code.", Path: filepath.Join(directory, "alpha", "SKILL.md")}}
	if !reflect.DeepEqual(skills, want) || len(skillErrors) != 3 {
		t.Fatalf("discovered skills = %#v, errors = %v", skills, skillErrors)
	}
	registry := NewRegistry(StaticTranslators{})
	for _, skill := range skills {
		if _, err := registry.RegisterSkill(skill); err != nil {
			t.Fatalf("discovered skill cannot be registered: %v", err)
		}
	}
}

func TestParseSkillFrontmatterYAML(t *testing.T) {
	for _, tt := range []struct {
		name, yaml, wantName, wantDescription string
	}{
		{"double quoted", `name: "review"
description: "Review: code # carefully" # comment`, "review", "Review: code # carefully"},
		{"single quoted", `name: 'review'
description: 'It''s a review'`, "review", "It's a review"},
		{"comments", "# metadata\nname: review # skill name\ndescription: Review code. # description", "review", "Review code."},
		{"plain continuation", "name: review\ndescription: Review code\n  and tests.", "review", "Review code and tests."},
		{"quoted continuation", "name: review\ndescription: \"Review code\n  and tests.\"", "review", "Review code and tests."},
		{"folded", "name: review\ndescription: >-\n  Review code\n  and tests.", "review", "Review code and tests."},
		{"literal", "name: review\ndescription: |\n  Review code.\n  Check tests.", "review", "Review code.\nCheck tests.\n"},
		{"ignored metadata", "metadata:\n  name: other\nname: review\ndescription: Review code.", "review", "Review code."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, newline := range []string{"\n", "\r\n"} {
				contents := strings.ReplaceAll("---\n"+tt.yaml+"\n---\nBody: [not YAML", "\n", newline)
				got, err := parseSkillFrontmatter([]byte(contents))
				if err != nil || got.Name != tt.wantName || got.Description != tt.wantDescription {
					t.Fatalf("metadata = %#v, error = %v", got, err)
				}
			}
		})
	}
}

func TestParseSkillFrontmatterRejectsMalformedYAML(t *testing.T) {
	for _, contents := range []string{
		"---\nname: [unterminated\ndescription: Review code.\n---\n",
		"---\nname: review\ndescription: [not, a, string]\n---\n",
		"---\nname: review\ndescription: Review code.\n",
	} {
		if _, err := parseSkillFrontmatter([]byte(contents)); err == nil {
			t.Fatalf("accepted malformed frontmatter %q", contents)
		}
	}
}

func TestDiscoverSkillsUsesDecodedNameForRegistrationAndDuplicates(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"alpha", "duplicate"} {
		path := filepath.Join(directory, name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("---\nname: 'review' # comment\ndescription: >-\n  Review code\n  and tests.\n---\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	skills, errs := DiscoverSkills(directory)
	if len(skills) != 1 || len(errs) != 1 || !strings.Contains(errs[0].Error(), "duplicate name") {
		t.Fatalf("skills = %#v, errors = %v", skills, errs)
	}
	r := NewRegistry(StaticTranslators{}).(*registry)
	if _, err := r.RegisterSkill(skills[0]); err != nil {
		t.Fatal(err)
	}
	if skill, ok := r.resolveSkill("review"); !ok || skill.Description != "Review code and tests." {
		t.Fatalf("decoded skill lookup = %#v, %v", skill, ok)
	}
}
