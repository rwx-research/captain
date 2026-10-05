<?php

function captainAttempt(string $id, bool $flaky = false): bool
{
    $path = getenv('CAPTAIN_PEST_ATTEMPTS');
    $attempts = file_exists($path) ? json_decode(file_get_contents($path), true) : [];
    $attempts[$id] = ($attempts[$id] ?? 0) + 1;
    file_put_contents($path, json_encode($attempts));

    return !$flaky || $attempts[$id] > 1;
}
