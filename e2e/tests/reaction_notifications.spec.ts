import {expect, test, type Page} from '@playwright/test';

import {api, login, newSession, TEAM, waitFor} from './helpers';

// Mattermost cuts plugin preference categories to 32 characters.
const CATEGORY = 'pp_dev.patika.reaction-notifications'.slice(0, 32);

async function setNotify(page: Page, value: string) {
    const me = await api(page, 'GET', '/users/me');
    await api(page, 'PUT', '/users/me/preferences', [{user_id: me.id, category: CATEGORY, name: 'notify', value}]);
}

// The admin's DM channel with the Reactions bot, and its recent bot posts.
async function botMessages(page: Page): Promise<string[]> {
    const me = await api(page, 'GET', '/users/me');
    const bot = await api(page, 'GET', '/users/username/reactions');
    const dm = await api(page, 'POST', '/channels/direct', [me.id, bot.id]);
    const posts = await api(page, 'GET', `/channels/${dm.id}/posts?per_page=30`);
    return Object.values<any>(posts.posts || {}).filter((p) => p.user_id === bot.id).map((p) => p.message);
}

test.describe('Reaction notifications', () => {
    test.beforeEach(async ({page}) => {
        await login(page, 'admin');
        await setNotify(page, 'all');
    });

    test('reactions to your message arrive as one grouped DM with a link', async ({page, browser}) => {
        const town = await api(page, 'GET', `/teams/name/${TEAM}/channels/name/town-square`);
        const text = `Campaign brief is ready ${Date.now()}`;
        const post = await api(page, 'POST', '/posts', {channel_id: town.id, message: text});

        const {context, page: mPage} = await newSession(browser, 'member');
        const member = await api(mPage, 'GET', '/users/me');
        for (const emoji of ['+1', 'tada']) {
            await api(mPage, 'POST', '/reactions', {user_id: member.id, post_id: post.id, emoji_name: emoji});
        }
        await context.close();

        const message = await waitFor(async () => (await botMessages(page)).find((m) => m.includes(text)));
        expect(message).toBe(`:+1: :tada: **@member** reacted to your message in ~town-square\n> ${text}\n\n[Open message](http://localhost:8068/_redirect/pl/${post.id})`);

        // It renders as a normal DM, and the link opens the message.
        await page.goto(`/${TEAM}/messages/@reactions`);
        const dmPost = page.locator('.post').filter({hasText: text}).last();
        await expect(dmPost).toContainText('reacted to your message in');
        await dmPost.getByRole('link', {name: 'Open message'}).click();
        await expect(page).toHaveURL(new RegExp(`/${TEAM}/pl/${post.id}|/${TEAM}/channels/town-square`));
    });

    test('a reaction taken back within the minute sends nothing', async ({page, browser}) => {
        const town = await api(page, 'GET', `/teams/name/${TEAM}/channels/name/town-square`);
        const undone = await api(page, 'POST', '/posts', {channel_id: town.id, message: `undo me ${Date.now()}`});
        const control = await api(page, 'POST', '/posts', {channel_id: town.id, message: `control ${Date.now()}`});

        const {context, page: mPage} = await newSession(browser, 'member');
        const member = await api(mPage, 'GET', '/users/me');
        await api(mPage, 'POST', '/reactions', {user_id: member.id, post_id: undone.id, emoji_name: 'smile'});
        await api(mPage, 'POST', '/reactions', {user_id: member.id, post_id: control.id, emoji_name: 'smile'});
        await mPage.waitForTimeout(3000); // a person takes a few seconds to change their mind
        await api(mPage, 'DELETE', `/users/${member.id}/posts/${undone.id}/reactions/smile`);
        await context.close();

        // Both groups are sent in the same round, so once the control arrives the other would have.
        await waitFor(async () => (await botMessages(page)).find((m) => m.includes(control.message)));
        expect((await botMessages(page)).some((m) => m.includes(undone.message))).toBe(false);
    });

    test('with notifications Off, reactions send nothing', async ({page, browser}) => {
        const town = await api(page, 'GET', `/teams/name/${TEAM}/channels/name/town-square`);
        const post = await api(page, 'POST', '/posts', {channel_id: town.id, message: `settings off ${Date.now()}`});
        await setNotify(page, 'off');

        const {context, page: mPage} = await newSession(browser, 'member');
        const member = await api(mPage, 'GET', '/users/me');
        await api(mPage, 'POST', '/reactions', {user_id: member.id, post_id: post.id, emoji_name: 'smile'});
        await context.close();

        // The setting is checked when the reaction arrives and again when it would be sent,
        // so stay Off past one full grouping window plus a send round.
        await page.waitForTimeout(100_000);
        expect((await botMessages(page)).some((m) => m.includes(post.message))).toBe(false);
    });

    test('Settings → Reaction Notifications saves the choice', async ({page}) => {
        await page.goto(`/${TEAM}/channels/town-square`);
        await page.getByRole('button', {name: 'Settings'}).first().click();
        const modal = page.getByRole('dialog').filter({hasText: 'Settings'});
        await modal.getByRole('tab', {name: /reaction notifications/i}).click();
        await modal.getByRole('tabpanel').getByRole('button', {name: /Edit/}).first().click();
        await modal.getByLabel('Only in direct and group messages').check();
        await modal.getByRole('button', {name: 'Save'}).click();

        await expect.poll(async () => {
            const prefs = await api<any[]>(page, 'GET', '/users/me/preferences');
            return prefs.find((p) => p.category === CATEGORY && p.name === 'notify')?.value;
        }).toBe('dms');
    });
});
