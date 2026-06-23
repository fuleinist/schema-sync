package schema

import (
	"testing"
)

func TestParseMySQLForeignKeys(t *testing.T) {
	// Actual MySQL SHOW CREATE TABLE output uses backtick-quoted identifiers
	createSQL := "CREATE TABLE `orders` (\n" +
		"  `id` int(11) NOT NULL AUTO_INCREMENT,\n" +
		"  `user_id` int(11) NOT NULL,\n" +
		"  `product_id` int(11) NOT NULL,\n" +
		"  PRIMARY KEY (`id`),\n" +
		"  CONSTRAINT `fk_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,\n" +
		"  CONSTRAINT `fk_product` FOREIGN KEY (`product_id`) REFERENCES `products` (`id`) ON DELETE RESTRICT ON UPDATE NO ACTION\n" +
		") ENGINE=InnoDB"

	fks := parseMySQLForeignKeys(createSQL)

	if len(fks) != 2 {
		t.Fatalf("expected 2 foreign keys, got %d", len(fks))
	}

	// First FK
	fk1 := fks[0]
	if fk1.Name != "fk_user" {
		t.Errorf("expected name 'fk_user', got '%s'", fk1.Name)
	}
	if len(fk1.Columns) != 1 || fk1.Columns[0] != "user_id" {
		t.Errorf("expected columns [user_id], got %v", fk1.Columns)
	}
	if fk1.RefTable != "users" {
		t.Errorf("expected ref table 'users', got '%s'", fk1.RefTable)
	}
	if len(fk1.RefColumns) != 1 || fk1.RefColumns[0] != "id" {
		t.Errorf("expected ref columns [id], got %v", fk1.RefColumns)
	}
	if fk1.OnDelete != "CASCADE" {
		t.Errorf("expected ON DELETE CASCADE, got '%s'", fk1.OnDelete)
	}
	if fk1.OnUpdate != "CASCADE" {
		t.Errorf("expected ON UPDATE CASCADE, got '%s'", fk1.OnUpdate)
	}

	// Second FK
	fk2 := fks[1]
	if fk2.Name != "fk_product" {
		t.Errorf("expected name 'fk_product', got '%s'", fk2.Name)
	}
	if fk2.RefTable != "products" {
		t.Errorf("expected ref table 'products', got '%s'", fk2.RefTable)
	}
	if fk2.OnDelete != "RESTRICT" {
		t.Errorf("expected ON DELETE RESTRICT, got '%s'", fk2.OnDelete)
	}
	if fk2.OnUpdate != "NO ACTION" {
		t.Errorf("expected ON UPDATE NO ACTION, got '%s'", fk2.OnUpdate)
	}
}

func TestParseMySQLForeignKeys_Composite(t *testing.T) {
	createSQL := "CREATE TABLE `order_items` (\n" +
		"  `order_id` int(11) NOT NULL,\n" +
		"  `product_id` int(11) NOT NULL,\n" +
		"  `quantity` int(11) NOT NULL,\n" +
		"  PRIMARY KEY (`order_id`, `product_id`),\n" +
		"  CONSTRAINT `fk_order_product` FOREIGN KEY (`order_id`, `product_id`) REFERENCES `orders` (`id`, `product_id`) ON DELETE CASCADE\n" +
		") ENGINE=InnoDB"

	fks := parseMySQLForeignKeys(createSQL)

	if len(fks) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(fks))
	}

	fk := fks[0]
	if fk.Name != "fk_order_product" {
		t.Errorf("expected name 'fk_order_product', got '%s'", fk.Name)
	}
	if len(fk.Columns) != 2 || fk.Columns[0] != "order_id" || fk.Columns[1] != "product_id" {
		t.Errorf("expected columns [order_id product_id], got %v", fk.Columns)
	}
	if len(fk.RefColumns) != 2 || fk.RefColumns[0] != "id" || fk.RefColumns[1] != "product_id" {
		t.Errorf("expected ref columns [id product_id], got %v", fk.RefColumns)
	}
	if fk.OnDelete != "CASCADE" {
		t.Errorf("expected ON DELETE CASCADE, got '%s'", fk.OnDelete)
	}
}

func TestParseMySQLForeignKeys_NoFKs(t *testing.T) {
	createSQL := "CREATE TABLE `simple` (\n" +
		"  `id` int(11) NOT NULL AUTO_INCREMENT,\n" +
		"  `name` varchar(255) NOT NULL,\n" +
		"  PRIMARY KEY (`id`)\n" +
		") ENGINE=InnoDB"

	fks := parseMySQLForeignKeys(createSQL)
	if len(fks) != 0 {
		t.Errorf("expected 0 foreign keys, got %d", len(fks))
	}
}

func TestParseMySQLForeignKeys_NoOnDeleteOnUpdate(t *testing.T) {
	createSQL := "CREATE TABLE `child` (\n" +
		"  `id` int(11) NOT NULL,\n" +
		"  `parent_id` int(11) NOT NULL,\n" +
		"  CONSTRAINT `fk_parent` FOREIGN KEY (`parent_id`) REFERENCES `parent` (`id`)\n" +
		") ENGINE=InnoDB"

	fks := parseMySQLForeignKeys(createSQL)
	if len(fks) != 1 {
		t.Fatalf("expected 1 foreign key, got %d", len(fks))
	}

	fk := fks[0]
	if fk.Name != "fk_parent" {
		t.Errorf("expected name 'fk_parent', got '%s'", fk.Name)
	}
	if fk.OnDelete != "" {
		t.Errorf("expected empty OnDelete, got '%s'", fk.OnDelete)
	}
	if fk.OnUpdate != "" {
		t.Errorf("expected empty OnUpdate, got '%s'", fk.OnUpdate)
	}
}
