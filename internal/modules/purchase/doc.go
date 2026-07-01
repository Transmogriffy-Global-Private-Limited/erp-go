package purchase

// Package purchase will own suppliers, purchase orders, goods receipt notes,
// and purchase lifecycle state.
//
// Purchase may emit receiving events that Inventory consumes.
// Purchase may emit payable events that Accounting consumes.
