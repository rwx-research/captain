<?php

test("handles / [a+b] (x)? $ ^ | \\ and ' quotes", function () {
    expect(captainAttempt('punctuation', true))->toBeTrue();
});

test("handles / [a+b] (x)? $ ^ | \\ and ' quotes extra", function () {
    expect(captainAttempt('similar'))->toBeTrue();
});

test('trailing spaces ', function () {
    expect(captainAttempt('trailing', true))->toBeTrue();
});

describe('outer', function () {
    describe('inner', function () {
        it('keeps Unicode → and :: separators', function () {
            expect(captainAttempt('nested', true))->toBeTrue();
        });
    });
});

test('named datasets', function (bool $flaky) {
    expect(captainAttempt($flaky ? 'named-failed' : 'named-passed', $flaky))->toBeTrue();
})->with([
    "failed / [a+b] ' quoted" => [true],
    'passing' => [false],
]);

test('numbered datasets', function (bool $flaky) {
    expect(captainAttempt($flaky ? 'numbered-failed' : 'numbered-passed', $flaky))->toBeTrue();
})->with([[true], [false]]);
