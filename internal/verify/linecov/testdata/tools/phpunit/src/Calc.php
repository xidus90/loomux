<?php
function one() { return 1; }
// thin wrapper around the extension, tested upstream
function two() { return 2; } // @codeCoverageIgnore
// platform shim, tested upstream
// @codeCoverageIgnoreStart
function three() { return 3; }
// @codeCoverageIgnoreEnd
function four() { return 4; }
function five() { return 5; } // @codeCoverageIgnore
