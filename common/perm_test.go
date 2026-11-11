package common

import "testing"

func TestIsRoot(t *testing.T) {
	if !IsRoot(RoleRootUser) {
		t.Fatal("RoleRootUser should be root")
	}
	for _, role := range []int{RoleGuestUser, RoleMemberUser, RoleVipMemberUser, RoleExternalMemberUser, 4, 5, 99} {
		if IsRoot(role) {
			t.Fatalf("role %d should not be root", role)
		}
	}
}

func TestPermForDefinedRoles(t *testing.T) {
	for _, role := range []int{RoleGuestUser, RoleMemberUser, RoleVipMemberUser, RoleExternalMemberUser} {
		if _, ok := PermFor(role); !ok {
			t.Fatalf("role %d should have permission", role)
		}
	}
}

func TestPermForOnlineLimits(t *testing.T) {
	cases := map[int]int64{
		RoleGuestUser:          1,
		RoleMemberUser:         1,
		RoleExternalMemberUser: 1,
		RoleVipMemberUser:      2,
	}
	for role, want := range cases {
		perm, ok := PermFor(role)
		if !ok {
			t.Fatalf("role %d missing", role)
		}
		if perm.MaxOnlineCount != want {
			t.Fatalf("role %d MaxOnlineCount=%d, want %d", role, perm.MaxOnlineCount, want)
		}
	}
}

func TestPermForUndefinedRoles(t *testing.T) {
	for _, role := range []int{4, 5, 99} {
		if _, ok := PermFor(role); ok {
			t.Fatalf("role %d should not have permission", role)
		}
	}
}

func TestPermForRoot(t *testing.T) {
	if _, ok := PermFor(RoleRootUser); ok {
		t.Fatal("root is not limited by RolePerm")
	}
}
