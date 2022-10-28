package features

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

// FeatureGate is feature gate 'manager' to be used
var fg *FeatureGate

// List of supported feature maturity levels
const (
	// Alpha - alpha version
	Alpha = string("ALPHA")

	// GA - general availability
	GA = string("GA")

	// Deprecated - feature that will be deprecated in 2 releases
	Deprecated = string("DEPRECATED")

	splitLength = 2
)

// List of supported features
const (
	// AlphaFeature - description
	AlphaFeature string = "alphaFeature"

	// BetaFeature - description
	BetaFeature string = "betaFeature"

	// GaFeature - description
	GaFeature string = "gaFeature"

	// DeprecatedFeature - description
	DeprecatedFeature string = "deprecatedFeature"
)

type featureSpec struct {
	// Default is the default enablement state for the feature
	Default bool
	// Maturity indicates the maturity level of the feature
	Maturity string
}

var defaultSriovDpFeatureGates = map[string]featureSpec{
	AlphaFeature:      {Default: false, Maturity: Alpha},
	GaFeature:         {Default: true, Maturity: GA},
	DeprecatedFeature: {Default: false, Maturity: Deprecated},
}

// FeatureGate defines FeatureGate structure
type FeatureGate struct {
	knownFeatures map[string]featureSpec
	enabled       map[string]bool
}

// NewFeatureGate creates new FeatureGate if it does not exist yet
func NewFeatureGate() {
	fg = newFeatureGate()
}

// GetFeatureGate returns current feature gate
func GetFeatureGate() (*FeatureGate, error) {
	var err error
	if fg == nil {
		err = fmt.Errorf("feature gate object was not initialized")
	}
	return fg, err
}

func newFeatureGate() *FeatureGate {
	if fg != nil {
		return fg
	}
	fg := &FeatureGate{}
	fg.knownFeatures = make(map[string]featureSpec)
	fg.enabled = make(map[string]bool)

	for k, v := range defaultSriovDpFeatureGates {
		fg.knownFeatures[k] = v
	}

	for k, v := range fg.knownFeatures {
		fg.enabled[k] = v.Default
	}
	return fg
}

// Enabled returns enabelement status of the provided feature
func (fg *FeatureGate) Enabled(featureName string) bool {
	return fg.enabled[featureName]
}

func (fg *FeatureGate) isFeatureSupported(featureName string) bool {
	_, exists := fg.knownFeatures[featureName]
	return exists
}

func (fg *FeatureGate) set(featureName string, status bool) error {
	if !fg.isFeatureSupported(featureName) {
		return fmt.Errorf("feature %s is not supported", featureName)
	}
	fg.enabled[featureName] = status
	if status && fg.knownFeatures[featureName].Maturity == Deprecated {
		glog.Warningf("WARNING: Feature %s will be deprecated soon", featureName)
	}
	return nil
}

// SetFromMap sets the enablement status of featuers accordig to a map
func (fg *FeatureGate) SetFromMap(valuesToSet map[string]bool) error {
	for k, v := range valuesToSet {
		if err := fg.set(k, v); err != nil {
			return err
		}
	}
	return nil
}

// SetFromString converts config string to map and sets the enablement status of the selected features
// copied from k8s and slightly changed - TBC?
func (fg *FeatureGate) SetFromString(value string) error {
	featureMap := make(map[string]bool)
	for _, s := range strings.Split(value, ",") {
		if s == "" {
			continue
		}
		splitted := strings.Split(s, "=")
		key := strings.TrimSpace(splitted[0])
		if len(splitted) != splitLength {
			if len(splitted) > splitLength {
				return fmt.Errorf("too many values for %s", key)
			}
			return fmt.Errorf("enablement value for %s is missing", key)
		}

		val := strings.TrimSpace(splitted[1])
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error while processing %s=%s, err: %v", key, val, err)
		}

		featureMap[key] = boolVal
	}
	return fg.SetFromMap(featureMap)
}
