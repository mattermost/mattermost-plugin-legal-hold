import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import AmazonS3BucketSettings from './amazon_s3_bucket_settings';

import Client from '@/client';

jest.mock('@/client', () => ({
    __esModule: true,
    default: {testAmazonS3Connection: jest.fn()},
}));

const SETTING_KEY = 'PluginSettings.Plugins.com+mattermost+plugin-legal-hold.amazons3bucketsettings';
const ID = 'com.mattermost.plugin-legal-hold';

const setup = (enable = false) => {
    const onChange = jest.fn();
    render(
        <AmazonS3BucketSettings
            value={{Enable: enable, Settings: {AmazonS3Bucket: 'bucket'} as never}}
            onChange={onChange}
        />,
    );
    return {onChange};
};

describe('AmazonS3BucketSettings', () => {
    test('disables the bucket fields until the custom bucket is enabled', async () => {
        const {onChange} = setup();
        const bucket = screen.getByRole('textbox', {name: 'Amazon S3 Bucket'});
        expect(bucket).toBeDisabled();
        expect(bucket).toHaveValue('bucket');
        expect(screen.getByRole('button', {name: 'Test connection'})).toBeDisabled();

        await userEvent.click(screen.getByTestId(`${ID}.EnableCustomS3Bucket-true`));

        expect(onChange).toHaveBeenCalledWith(SETTING_KEY, expect.objectContaining({Enable: true}));
        expect(bucket).toBeEnabled();
        expect(screen.getByRole('button', {name: 'Test connection'})).toBeEnabled();
    });

    test('reports a changed field to the admin console', async () => {
        const {onChange} = setup(true);

        await userEvent.type(screen.getByRole('textbox', {name: 'Amazon S3 Region'}), 'eu');

        expect(onChange).toHaveBeenLastCalledWith(SETTING_KEY, expect.objectContaining({
            Settings: expect.objectContaining({AmazonS3Region: 'eu'}),
        }));
    });

    test('masks an existing secret key', () => {
        render(
            <AmazonS3BucketSettings
                value={{Enable: true, Settings: {AmazonS3SecretAccessKey: 'real'} as never}}
                onChange={jest.fn()}
            />,
        );

        expect(screen.getByRole('textbox', {name: 'Amazon S3 Secret Access Key'})).toHaveValue('*'.repeat(32));
    });

    test('shows the result of a successful connection test', async () => {
        (Client.testAmazonS3Connection as jest.Mock).mockResolvedValue({message: 'Connected'});
        setup(true);

        await userEvent.click(screen.getByRole('button', {name: 'Test connection'}));

        expect(await screen.findByRole('status')).toHaveTextContent('Connected');
    });

    test('shows the error of a failed connection test', async () => {
        (Client.testAmazonS3Connection as jest.Mock).mockRejectedValue(new Error('access denied'));
        setup(true);

        await userEvent.click(screen.getByRole('button', {name: 'Test connection'}));

        expect(await screen.findByRole('alert')).toHaveTextContent('access denied');
    });

    test('asks to save unsaved changes before it tests the connection', async () => {
        const save = document.createElement('button');
        save.id = 'saveSetting';
        document.body.appendChild(save);
        setup(true);

        await userEvent.click(screen.getByRole('button', {name: 'Test connection'}));

        expect(await screen.findByRole('alert')).toHaveTextContent('Save the settings before you test the connection.');
        expect(Client.testAmazonS3Connection).not.toHaveBeenCalled();
        save.remove();
    });
});
