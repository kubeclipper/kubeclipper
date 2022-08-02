package imageproxy

import (
	"fmt"
	"net/url"

	"github.com/spf13/pflag"
)

type Options struct {
	KcImageRepoMirror string `json:"kcImageRepoMirror" yaml:"kcImageRepoMirror"`
}

func NewOptions() *Options {
	return &Options{
		KcImageRepoMirror: "",
	}
}

func (s *Options) Validate() []error {
	var errs []error
	uri, err := url.Parse(s.KcImageRepoMirror)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	if uri.Scheme != "" {
		errs = append(errs, fmt.Errorf("kc image repo mirror is not support incoming protcol: %s", uri.Scheme))
	}
	return errs
}

func (s *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.KcImageRepoMirror, "kc-image-repo-mirror", s.KcImageRepoMirror, "K8s image repository mirror")
}
