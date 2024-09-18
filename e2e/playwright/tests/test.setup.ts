import {test as setup} from '@e2e-support/test_fixture';

setup('ensure plugin is enabled', async ({pw}) => {
    const {adminClient} = await pw.getAdminClient();
    await adminClient.enablePlugin('com.mattermost.plugin-legal-hold');
});
