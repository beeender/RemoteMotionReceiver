-- complain if script is sourced in psql, rather than via CREATE EXTENSION
\echo Use "CREATE EXTENSION remote_motion" to load this file. \quit

CREATE OR REPLACE FUNCTION count_prime(integer) RETURNS int
AS '$libdir/remote_motion'
LANGUAGE C IMMUTABLE STRICT;

CREATE OR REPLACE FUNCTION count_prime_RMOTION(a integer) RETURNS integer AS $$
# container: runtime
    return a
$$ LANGUAGE plpython3u;
