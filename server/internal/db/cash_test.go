package db

import "testing"

func TestMallLoadsProductListWithNativeQuantities(t *testing.T) {
	pool, err := Pool(GameDefs)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(`INSERT INTO PS_GameDefs_ProductList(ProductName,ProductCode,BuyCost,ItemID1,ItemCount1,ItemID24,ItemCount24) VALUES('SQL native','SQL_NATIVE_TEST',0,25001,255,1001,1)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(`DELETE FROM PS_GameDefs_ProductList WHERE ProductCode='SQL_NATIVE_TEST'`) })
	defs, err := loadDefs()
	if err != nil {
		t.Fatal(err)
	}
	p := defs.Products["SQL_NATIVE_TEST"]
	if p == nil || p.BuyCost != 0 || len(p.Items) != 2 || p.Items[0].Count != 255 || p.Items[1].Count != 1 {
		t.Fatal("ProductList lost quantities/last slot", p)
	}
}
