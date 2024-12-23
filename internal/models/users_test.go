package models

import (
	"snippetbox/internal/assert"
	"testing"
)

func TestUserModel_Exists(t *testing.T) {
	//如果提供“-short”标志跳过此集成测试
	if testing.Short() {
		t.Skip("model:skipping integration test")
	}
	tests := []struct {
		name   string
		userID int
		want   bool
	}{
		{
			name:   "Valid ID",
			userID: 1,
			want:   true,
		},
		{
			name:   "Zero ID",
			userID: 0,
			want:   false,
		},
		{
			name:   "Non-existent ID",
			userID: 2,
			want:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := newTestDB(t)
			m := UserModel{DB: db}
			exists, err := m.Exists(test.userID)
			assert.Equal(t, exists, test.want)
			assert.NilError(t, err)
		})
	}
}
