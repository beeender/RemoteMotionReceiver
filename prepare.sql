create table t1 (id int);
insert into t1 select * from generate_series(100000, 105000);

CREATE OR REPLACE FUNCTION count_prime(integer) RETURNS int
AS '$libdir/overhead'
LANGUAGE C IMMUTABLE STRICT;

CREATE OR REPLACE FUNCTION count_prime_RMOTION(a integer) RETURNS integer AS $$
    return a
$$ LANGUAGE plpython3u;
