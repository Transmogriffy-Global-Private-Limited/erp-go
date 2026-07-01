package accounting

// Package accounting will own ledgers, journals, vouchers, posting rules,
// and accounting reports.
//
// Other modules must not directly write final accounting journal tables.
// They emit events or request accounting commands.
