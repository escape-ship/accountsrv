START TRANSACTION;

CREATE TABLE escape.payments (
    id CHAR(36) NOT NULL DEFAULT (UUID()),
    ref_order_id CHAR(36) NOT NULL,
    name CHAR(36) NOT NULL,
    loyalty_member_id CHAR(36) NOT NULL,
    order_status INT NOT NULL,
    updated TIMESTAMP NULL,
    created TIMESTAMP NULL,
    CONSTRAINT pk_payments PRIMARY KEY (id)
);

COMMIT;