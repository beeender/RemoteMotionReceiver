MODULES = count_prime
EXTENSION = remote_motion
DATA = remote_motion--1.0.0.sql

PG_CONFIG = pg_config
PGXS := $(shell $(PG_CONFIG) --pgxs)
include $(PGXS)
