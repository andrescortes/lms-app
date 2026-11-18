package domain

import "fmt"

// Profile is a learner's personal record, linked one-to-one with a User. It is
// created empty at registration and populated by a later feature.
type Profile struct {
	UserID         string
	NativeLanguage *string
	Goals          *string
	Timezone       *string
	AvatarURL      *string
}

// NewEmptyProfile returns a profile owned by userID with every optional field
// unset, matching the registration invariant.
func NewEmptyProfile(userID string) Profile {
	return Profile{UserID: userID}
}

// Validate enforces that the profile has an owning user.
func (p Profile) Validate() error {
	if p.UserID == "" {
		return fmt.Errorf("profile user id is empty: %w", ErrInvalidProfile)
	}
	return nil
}
