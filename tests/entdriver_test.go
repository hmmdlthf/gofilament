package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/golaravel/gofilament/pkg/entdriver"
	"github.com/golaravel/gofilament/pkg/filament"
)

// Mock Ent Model
type MockUser struct {
	ID    int
	Name  string
	Email string
	Role  string
}

// Mock Ent Query Builder
type MockUserQuery struct {
	items  []*MockUser
	limit  int
	offset int
}

func (q *MockUserQuery) Count(ctx context.Context) (int, error) {
	return len(q.items), nil
}

func (q *MockUserQuery) Limit(limit int) *MockUserQuery {
	q.limit = limit
	return q
}

func (q *MockUserQuery) Offset(offset int) *MockUserQuery {
	q.offset = offset
	return q
}

func (q *MockUserQuery) All(ctx context.Context) ([]*MockUser, error) {
	start := q.offset
	if start > len(q.items) {
		return nil, nil
	}
	end := start + q.limit
	if end > len(q.items) || q.limit == 0 {
		end = len(q.items)
	}
	return q.items[start:end], nil
}

// Mock Ent Create Builder
type MockUserCreate struct {
	user *MockUser
}

func (c *MockUserCreate) SetName(name string) *MockUserCreate {
	c.user.Name = name
	return c
}

func (c *MockUserCreate) SetEmail(email string) *MockUserCreate {
	c.user.Email = email
	return c
}

func (c *MockUserCreate) SetRole(role string) *MockUserCreate {
	c.user.Role = role
	return c
}

func (c *MockUserCreate) Save(ctx context.Context) (*MockUser, error) {
	return c.user, nil
}

// Mock Ent Update Builder
type MockUserUpdateOne struct {
	user *MockUser
}

func (u *MockUserUpdateOne) SetName(name string) *MockUserUpdateOne {
	u.user.Name = name
	return u
}

func (u *MockUserUpdateOne) SetEmail(email string) *MockUserUpdateOne {
	u.user.Email = email
	return u
}

func (u *MockUserUpdateOne) SetRole(role string) *MockUserUpdateOne {
	u.user.Role = role
	return u
}

func (u *MockUserUpdateOne) Save(ctx context.Context) (*MockUser, error) {
	return u.user, nil
}

// Mock Ent Delete Builder
type MockUserDeleteOne struct {
	id      int
	deleted bool
}

func (d *MockUserDeleteOne) Exec(ctx context.Context) error {
	d.deleted = true
	return nil
}

// Mock Ent UserClient
type MockUserClient struct {
	store map[int]*MockUser
}

func (c *MockUserClient) Query() *MockUserQuery {
	var list []*MockUser
	for _, u := range c.store {
		list = append(list, u)
	}
	return &MockUserQuery{items: list}
}

func (c *MockUserClient) Get(ctx context.Context, id int) (*MockUser, error) {
	if u, ok := c.store[id]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (c *MockUserClient) Create() *MockUserCreate {
	newUser := &MockUser{ID: len(c.store) + 1}
	c.store[newUser.ID] = newUser
	return &MockUserCreate{user: newUser}
}

func (c *MockUserClient) UpdateOneID(id int) *MockUserUpdateOne {
	u := c.store[id]
	return &MockUserUpdateOne{user: u}
}

func (c *MockUserClient) DeleteOneID(id int) *MockUserDeleteOne {
	delete(c.store, id)
	return &MockUserDeleteOne{id: id}
}

func TestEntDriverZeroBoilerplateAdapter(t *testing.T) {
	client := &MockUserClient{
		store: map[int]*MockUser{
			1: {ID: 1, Name: "Alice", Email: "alice@example.com", Role: "admin"},
			2: {ID: 2, Name: "Bob", Email: "bob@example.com", Role: "editor"},
		},
	}

	res := filament.NewResource[*MockUser]("users")

	// 1. Bind resource to Ent client with zero boilerplate!
	entdriver.Bind(res, entdriver.Config[*MockUser]{
		Client: client,
	})

	ctx := filament.NewContext(nil, filament.NewPanel("admin"))

	// 2. Test OnList
	records, total, err := res.GetOnList()(ctx, "", "", "asc", 1, 10, nil)
	if err != nil {
		t.Fatalf("OnList failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got: %d", total)
	}
	if len(records) != 2 {
		t.Errorf("Expected 2 records, got: %d", len(records))
	}

	// 3. Test OnFind
	user1, err := res.GetOnFind()(ctx, "1")
	if err != nil {
		t.Fatalf("OnFind failed: %v", err)
	}
	if user1.Name != "Alice" {
		t.Errorf("Expected name 'Alice', got: %s", user1.Name)
	}

	// 4. Test OnCreate
	err = res.GetOnCreate()(ctx, map[string]any{
		"name":  "Charlie",
		"email": "charlie@example.com",
		"role":  "viewer",
	})
	if err != nil {
		t.Fatalf("OnCreate failed: %v", err)
	}
	if len(client.store) != 3 {
		t.Errorf("Expected 3 users in store, got: %d", len(client.store))
	}
	if client.store[3].Name != "Charlie" || client.store[3].Email != "charlie@example.com" {
		t.Errorf("User 3 not created properly: %+v", client.store[3])
	}

	// 5. Test OnUpdate
	err = res.GetOnUpdate()(ctx, "3", map[string]any{
		"name": "Charlie Senior",
	})
	if err != nil {
		t.Fatalf("OnUpdate failed: %v", err)
	}
	if client.store[3].Name != "Charlie Senior" {
		t.Errorf("Expected name 'Charlie Senior', got: %s", client.store[3].Name)
	}

	// 6. Test OnDelete
	err = res.GetOnDelete()(ctx, "3")
	if err != nil {
		t.Fatalf("OnDelete failed: %v", err)
	}
	if _, ok := client.store[3]; ok {
		t.Errorf("Expected user 3 to be deleted")
	}

	// 7. Test OnDeleteBulk
	err = res.GetOnDeleteBulk()(ctx, []string{"1", "2"})
	if err != nil {
		t.Fatalf("OnDeleteBulk failed: %v", err)
	}
	if len(client.store) != 0 {
		t.Errorf("Expected all users to be deleted, got: %d", len(client.store))
	}
}
