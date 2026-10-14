import {defineConfig} from '@playwright/test';

// Reactions are grouped for a minute before the DM is sent, so tests wait for the real
// delivery (about 60–80 seconds) instead of faking the clock.
export default defineConfig({
    testDir: './tests',
    timeout: 240_000,
    workers: 1,
    retries: 0,
    reporter: [['list']],
    use: {
        baseURL: 'http://localhost:8068',
        viewport: {width: 1400, height: 900},
        timezoneId: 'Europe/Istanbul',
        channel: 'chrome',
    },
});
