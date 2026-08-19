package testharness

var SchemaFixture = Fixture{
	Name: "schema-fixture",
	Setup: []string{
		`CREATE TABLE IF NOT EXISTS ghostfleet_harness_probe (id INTEGER PRIMARY KEY, note TEXT NOT NULL)`,
		`INSERT INTO ghostfleet_harness_probe (id, note) VALUES (1, 'fixture')`,
	},
	Cleanup: []string{
		`DROP TABLE IF EXISTS ghostfleet_harness_probe`,
	},
}
