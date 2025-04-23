package update

import "regexp"

var Version = "dev" // set by ldflags during release

type semVer string

var semVerRegex = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

func (v semVer) Split() []string {
	if v == "dev" {
		return []string{"999", "0", "0"}
	}
	a := semVerRegex.FindStringSubmatch(string(v))
	if a == nil {
		return nil
	}
	return a[1:]
}

func (v semVer) IsNewerThan(other semVer) bool {
	a := v.Split()
	b := other.Split()

	if a == nil || b == nil {
		return false
	}

	for i := 0; i <= 2; i++ {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}

	return false
}
