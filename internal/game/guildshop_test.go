package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShopItemsFollowTheMarketUpToTheDeepestFloor(t *testing.T) {
	g := New(27182818, Warrior)
	gd := g.Guild
	gd.Gold = 1 << 40
	if gd.ShopLevel() != 1 || gd.Build(Market) {
		t.Fatalf("new guild: shop level %d, market built %v", gd.ShopLevel(), gd.Buildings[Market])
	}
	g.MaxDepth = 25
	for gd.Build(Market) {
	}
	if gd.Buildings[Market] != 2 || gd.ShopLevel() != 25 || !gd.MarketLocked() {
		t.Fatalf("market %d shop level %d", gd.Buildings[Market], gd.ShopLevel())
	}
	gd.refreshShop()
	for _, it := range gd.Shop {
		if it.Floor != 25 {
			t.Fatalf("item from floor %d", it.Floor)
		}
	}
}

func TestShopRenewsEveryHourAndForAPrice(t *testing.T) {
	g := New(16180339, Warrior)
	gd := g.Guild
	first := gd.Shop[0].ID
	gd.Advance(ShopRefreshMs - 1)
	if gd.Shop[0].ID != first {
		t.Fatal("shop renewed early")
	}
	gd.Advance(1)
	second := gd.Shop[0].ID
	if second == first {
		t.Fatal("shop did not renew after an hour")
	}
	if gd.Restock() || gd.Shop[0].ID != second {
		t.Fatal("restocked without gold")
	}
	gd.Gold = gd.RestockPrice()
	if !gd.Restock() || gd.Gold != 0 || gd.Shop[0].ID == second {
		t.Fatal("paid restock failed")
	}
}

func TestShopSellsToHeroesInTheGuildWhoCanUseIt(t *testing.T) {
	g := New(57721566, Warrior)
	gd := g.Guild
	gd.Gold = 1 << 30
	gd.Shop = []*Item{{Slot: Shield, Base: 3}}
	mage := New(1, Mage)
	mage.Guild = gd
	gd.Heroes = append(gd.Heroes, mage)
	if mage.Refuse(gd.Shop[0]) != ErrOtherUse {
		t.Fatal("mage may take a shield")
	}
	g.Depart()
	if g.Refuse(gd.Shop[0]) != ErrAway {
		t.Fatal("hero away may take an item")
	}
	g.BuyShop(0)
	if len(gd.Shop) != 1 || g.Hero.Equip[Shield] != nil {
		t.Fatal("hero away bought an item")
	}
}

func TestBagAndChestBelongToTheGuild(t *testing.T) {
	g := departed(14142135)
	gd := g.Guild
	gd.Gold = 1 << 30
	gd.Stash = []*Item{{Slot: Boots, Base: 2}}
	if !gd.UpgradeBag() || g.BagCap() != BagSize+BagSlotsPerLevel {
		t.Fatalf("bag %d", g.BagCap())
	}
	gd.SellStash(0)
	if len(gd.Stash) != 0 {
		t.Fatal("chest sale needed a hero at home")
	}
}

func TestV12SaveGivesTheGuildTheBiggestBag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	g := New(31415926, Warrior)
	other := New(2, Mage)
	other.Guild = g.Guild
	g.Guild.Heroes = append(g.Guild.Heroes, other)
	g.BagLevel, other.BagLevel = 2, 5
	raw, _ := json.Marshal(g.Guild)
	var gd map[string]any
	json.Unmarshal(raw, &gd)
	delete(gd, "Shop")
	data, _ := json.Marshal(map[string]any{"Version": 12, "SavedAt": time.Now(), "Guild": gd})
	os.WriteFile(path, data, 0o644)
	got, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Guild.BagLevel != 5 || got.BagLevel != 0 || len(got.Guild.Shop) != ShopSize {
		t.Fatalf("guild bag %d hero bag %d shop %d", got.Guild.BagLevel, got.BagLevel, len(got.Guild.Shop))
	}
}
