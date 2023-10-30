package release

import "fmt"

const repo = "tetrateio/tis-archives"

type Assets string

func (a Assets) Tag() string {
	return ""
}

func Create(a Assets) error {
	fmt.Println(repo)
	// if err := sh.RunV("gh", "release", "view", a.Tag(), "-R", repo); err != nil {
	// 	if err := sh.RunV("gh", "release", "create", a.Tag(), string(a), "-n", a.Tag(), "-t", a.Tag(), "-R", repo); err == nil {
	// 		return err
	// 	}
	// } else {
	// 	return sh.RunV("gh", "release", "upload", a.Tag(), string(a), "--clobber", "-R", repo)
	// }
	return nil
}
