package service

import (
	"testing"
)

func TestGetClients(t *testing.T) {
	t.Run("房间存在", func(t *testing.T) {
		roomMgr := NewRoomManager()
		_ = roomMgr.Join("room1", "test1")
		got, err := roomMgr.GetClients("room1")
		if err != nil {
			t.Fatalf("GetClients发生错误: %v", err)
		}
		if len(got) != 1 || got[0] != "test1" {
			t.Errorf("期望 [test1], 得到 %v", got)
		}
	})
	t.Run("房间不存在", func(t *testing.T) {
		roomMgr := NewRoomManager()
		got, err := roomMgr.GetClients("room1")
		if err != nil {
			t.Fatalf("GetClients发生错误: %v", err)
		}
		if got != nil {
			t.Errorf("期望 nil, 得到 %v", got)
		}
	})
}

func TestHasRoom(t *testing.T) {
	t.Run("房间存在", func(t *testing.T) {
		roomMgr := NewRoomManager()
		_ = roomMgr.Join("room1", "test1")
		exist, err := roomMgr.HasRoom("room1")
		if err != nil {
			t.Fatalf("HasRoom发生错误: %v", err)
		}
		if !exist {
			t.Error("期望 HasRoom 返回 true, 实际 false")
		}
	})
	t.Run("房间不存在", func(t *testing.T) {
		roomMgr := NewRoomManager()
		exist, err := roomMgr.HasRoom("room1")
		if err != nil {
			t.Fatalf("HasRoom发生错误: %v", err)
		}
		if exist {
			t.Error("期望 HasRoom 返回 false, 实际 true")
		}
	})
}

func TestJoin(t *testing.T) {
	tests := []struct {
		name     string
		roomIDs  []string
		clientID string
	}{
		{"正常加入房间", []string{"room1"}, "test1"},
		{"重复加入房间", []string{"room1"}, "test1"},
		{"加入多个房间", []string{"room1", "room2"}, "test1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			roomMgr := NewRoomManager()
			for _, roomID := range test.roomIDs {
				err := roomMgr.Join(roomID, test.clientID)
				if err != nil {
					t.Fatalf("Join发生错误: %v", err)
				}
			}
			// 验证每个房间都存在且包含该客户端
			for _, roomID := range test.roomIDs {
				exist, err := roomMgr.HasRoom(roomID)
				if err != nil {
					t.Fatalf("HasRoom发生错误: %v", err)
				}
				if !exist {
					t.Errorf("期望房间 %s 存在, 实际不存在", roomID)
				}
				clients, err := roomMgr.GetClients(roomID)
				if err != nil {
					t.Fatalf("GetClients发生错误: %v", err)
				}
				if len(clients) != 1 || clients[0] != test.clientID {
					t.Errorf("期望房间 %s 包含 %s, 实际 %v", roomID, test.clientID, clients)
				}
			}
		})
	}
}

func TestLeave(t *testing.T) {
	t.Run("正常离开房间", func(t *testing.T) {
		roomMgr := NewRoomManager()
		_ = roomMgr.Join("room1", "test1")
		_ = roomMgr.Join("room1", "test2")
		err := roomMgr.Leave("room1", "test1")
		if err != nil {
			t.Fatalf("Leave发生错误: %v", err)
		}
		clients, err := roomMgr.GetClients("room1")
		if err != nil {
			t.Fatalf("GetClients发生错误: %v", err)
		}
		if len(clients) != 1 || clients[0] != "test2" {
			t.Errorf("期望 room1 中仅剩 test2, 实际 %v", clients)
		}
	})
	t.Run("离开不存在的房间", func(t *testing.T) {
		roomMgr := NewRoomManager()
		_ = roomMgr.Join("room1", "test1")
		err := roomMgr.Leave("room999", "test1") // 房间不存在，应无操作
		if err != nil {
			t.Fatalf("Leave发生错误: %v", err)
		}
		clients, err := roomMgr.GetClients("room1")
		if err != nil {
			t.Fatalf("GetClients发生错误: %v", err)
		}
		if len(clients) != 1 || clients[0] != "test1" {
			t.Errorf("期望 test1 仍在 room1, 实际 %v", clients)
		}
	})
	t.Run("离开后房间无人自动删除", func(t *testing.T) {
		roomMgr := NewRoomManager()
		_ = roomMgr.Join("room1", "test1")
		err := roomMgr.Leave("room1", "test1")
		if err != nil {
			t.Fatalf("Leave发生错误: %v", err)
		}
		exist, err := roomMgr.HasRoom("room1")
		if err != nil {
			t.Fatalf("HasRoom发生错误: %v", err)
		}
		if exist {
			t.Error("期望 room1 已被删除, 但存在")
		}
		clients, err := roomMgr.GetClients("room1")
		if err != nil {
			t.Fatalf("GetClients发生错误: %v", err)
		}
		if len(clients) != 0 {
			t.Error("期望空房间的 GetClients 返回空切片")
		}
	})
}

func TestRemoveClientFromAllRooms(t *testing.T) {
	type testCase struct {
		name                 string
		roomIDs              []string
		clientIDs            []string
		removeClientID       string
		multiClientMultiRoom bool // 标记多客户端多房间场景
	}
	tests := []testCase{
		{
			name:           "客户端在一个房间内",
			roomIDs:        []string{"room1"},
			clientIDs:      []string{"test1"},
			removeClientID: "test1",
		},
		{
			name:           "客户端在多个房间内",
			roomIDs:        []string{"room1", "room2"},
			clientIDs:      []string{"test1"},
			removeClientID: "test1",
		},
		{
			name:           "客户端不在任何房间内",
			roomIDs:        []string{"room1"},
			clientIDs:      []string{"test1"},
			removeClientID: "test999",
		},
		{
			name:                 "多个客户端多个房间",
			roomIDs:              []string{"room1", "room2"},
			clientIDs:            []string{"test1", "test2"},
			removeClientID:       "", // 本场景分两次移除，不直接使用
			multiClientMultiRoom: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			roomMgr := NewRoomManager()
			if test.multiClientMultiRoom {
				// 多客户端多房间：test1 在 room1, test2 在 room2
				_ = roomMgr.Join("room1", "test1")
				_ = roomMgr.Join("room2", "test2")

				err := roomMgr.RemoveClientFromAllRooms("test1")
				if err != nil {
					t.Fatalf("RemoveClientFromAllRooms错误:%v", err)
				}
				err = roomMgr.RemoveClientFromAllRooms("test2")
				if err != nil {
					t.Fatalf("RemoveClientFromAllRooms错误:%v", err)
				}

				// 验证两个房间都应为空或不存在
				exist, err := roomMgr.HasRoom("room1")
				if err != nil {
					t.Fatal(err)
				}
				if exist {
					t.Error("期望 room1 被删除, 但存在")
				}
				exist, err = roomMgr.HasRoom("room2")
				if err != nil {
					t.Fatal(err)
				}
				if exist {
					t.Error("期望 room2 被删除, 但存在")
				}

				clients, err := roomMgr.GetClients("room1")
				if err != nil {
					t.Fatal(err)
				}
				if len(clients) != 0 {
					t.Error("期望 room1 的客户端列表为空")
				}
				clients, err = roomMgr.GetClients("room2")
				if err != nil {
					t.Fatal(err)
				}
				if len(clients) != 0 {
					t.Error("期望 room2 的客户端列表为空")
				}
				return
			}
			// 普通场景：将所有 clientIDs 加入所有 roomIDs
			for _, roomID := range test.roomIDs {
				for _, clientID := range test.clientIDs {
					err := roomMgr.Join(roomID, clientID)
					if err != nil {
						t.Fatalf("Join错误:%v", err)
					}
				}
			}
			err := roomMgr.RemoveClientFromAllRooms(test.removeClientID)
			if err != nil {
				t.Fatalf("RemoveClientFromAllRooms错误:%v", err)
			}
			// 验证被移除的客户端确实不在任何房间中
			for _, roomID := range test.roomIDs {
				clients, err := roomMgr.GetClients(roomID)
				if err != nil {
					t.Fatal(err)
				}
				for _, c := range clients {
					if c == test.removeClientID {
						t.Errorf("客户端 %s 应被移除, 但仍在房间 %s 中", test.removeClientID, roomID)
					}
				}
			}
			// 如果移除的客户端是房间内唯一成员，房间应被自动删除
			if len(test.clientIDs) == 1 && test.removeClientID == test.clientIDs[0] {
				for _, roomID := range test.roomIDs {
					exist, err := roomMgr.HasRoom(roomID)
					if err != nil {
						t.Fatal(err)
					}
					if exist {
						t.Errorf("房间 %s 应为空但未被删除", roomID)
					}
				}
			}
		})
	}
}

func TestGetAllRoomsConnStats(t *testing.T) {
	t.Run("存在房间", func(t *testing.T) {
		roomMgr := NewRoomManager()
		_ = roomMgr.Join("room1", "test1")
		_ = roomMgr.Join("room2", "test2")
		got, err := roomMgr.GetAllRoomsConnStats()
		if err != nil {
			t.Fatalf("GetAllRoomsConnStats错误:%v", err)
		}
		if got["room1"] != 1 {
			t.Errorf("期望 room1 连接数 1, 得到 %d", got["room1"])
		}
		if got["room2"] != 1 {
			t.Errorf("期望 room2 连接数 1, 得到 %d", got["room2"])
		}
	})
	t.Run("不存在房间", func(t *testing.T) {
		roomMgr := NewRoomManager()
		got, err := roomMgr.GetAllRoomsConnStats()
		if err != nil {
			t.Fatalf("GetAllRoomsConnStats错误:%v", err)
		}
		if len(got) != 0 {
			t.Errorf("期望空 map, 得到 %v", got)
		}
	})
}
