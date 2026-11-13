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

func TestDefaultServerMemory(t *testing.T) {
	guest, ok := PermFor(RoleGuestUser)
	if !ok {
		t.Fatal("guest permission missing")
	}
	xms, xmx := DefaultServerMemory(guest.ServerMaxMem, false)
	if xmx != guest.ServerMaxMem {
		t.Fatalf("guest xmx=%d, want %d", xmx, guest.ServerMaxMem)
	}
	if xms > xmx {
		t.Fatalf("guest xms=%d > xmx=%d", xms, xmx)
	}

	member, ok := PermFor(RoleMemberUser)
	if !ok {
		t.Fatal("member permission missing")
	}
	xms, xmx = DefaultServerMemory(member.ServerMaxMem, false)
	if xmx != 2*GB || xms != 1*GB {
		t.Fatalf("member defaults xms=%d xmx=%d", xms, xmx)
	}

	xms, xmx = DefaultServerMemory(0, true)
	if xmx != 2*GB || xms != 1*GB {
		t.Fatalf("unlimited defaults xms=%d xmx=%d", xms, xmx)
	}
}
