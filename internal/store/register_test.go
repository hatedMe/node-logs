package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMaxAppID(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	next, err := s.MaxAppID(ctx)
	if err != nil {
		t.Fatalf("MaxAppID 失败: %v", err)
	}
	if next != 800799 {
		t.Errorf("空库时 MaxAppID 应为 800799（下一可用 800800），实际为 %d", next)
	}

	ensureApp(t, s, "800800")
	ensureApp(t, s, "800805")
	next, err = s.MaxAppID(ctx)
	if err != nil {
		t.Fatalf("MaxAppID 失败: %v", err)
	}
	if next != 800805 {
		t.Errorf("MaxAppID 应为 800805，实际为 %d", next)
	}
}

func TestUpdateUserPassword(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := s.CreateUser(ctx, "olduser", "E10ADC3949BA59ABBE56E057F20F883E"); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if err := s.UpdateUserPassword(ctx, "olduser", "new-bcrypt-hash"); err != nil {
		t.Fatalf("更新密码失败: %v", err)
	}
	got, err := s.GetUserByUsername(ctx, "olduser")
	if err != nil || got.PasswordHash != "new-bcrypt-hash" {
		t.Fatalf("密码未更新: (%v, %v)", got, err)
	}
	if err := s.UpdateUserPassword(ctx, "ghost", "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("未知用户应返回 ErrNotFound，实际为 %v", err)
	}
}

func TestTxCreateUser(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.WithTx(ctx, func(tx *Tx) error {
		u, err := tx.CreateUser(ctx, "txuser", "hash-x")
		if err != nil {
			return err
		}
		if u.ID == 0 || u.Username != "txuser" {
			t.Errorf("事务内创建用户返回错误: %+v", u)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx 创建用户失败: %v", err)
	}
	got, err := s.GetUserByUsername(ctx, "txuser")
	if err != nil || got.PasswordHash != "hash-x" {
		t.Fatalf("事务提交后应可读到用户: (%v, %v)", got, err)
	}
}
