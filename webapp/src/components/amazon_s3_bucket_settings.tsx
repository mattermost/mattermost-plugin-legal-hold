import React, {useMemo, useState} from 'react';

import {IntlProvider} from 'react-intl';

import Client from '@/client';

import BooleanSetting from './admin_console_settings/boolean_setting';
import TextSetting from './admin_console_settings/text_setting';
import SaveButton from './mattermost-webapp/save_button';
import BaseSetting from './admin_console_settings/base_setting';
import StatusMessage from './admin_console_settings/status_message';

type FileSettings = {
    DriverName: string;
    AmazonS3RequestTimeoutMilliseconds: number;
    AmazonS3Bucket: string;
    AmazonS3PathPrefix: string;
    AmazonS3Region: string;
    AmazonS3Endpoint: string;
    AmazonS3AccessKeyId: string;
    AmazonS3SecretAccessKey: string;
    AmazonS3SSL: boolean;
    AmazonS3SSE: boolean;
};

type AmazonS3BucketSettingsData = {
    Enable: boolean;
    Settings: FileSettings;
};

const useS3BucketForm = (initialValue: AmazonS3BucketSettingsData | undefined, onChange: (id: string, value: AmazonS3BucketSettingsData) => void) => {
    const [formState, setFormState] = useState<AmazonS3BucketSettingsData>({
        Enable: initialValue?.Enable ?? false,
        Settings: {
            DriverName: 'amazons3',
            AmazonS3RequestTimeoutMilliseconds: 30000,
            AmazonS3Bucket: '',
            AmazonS3PathPrefix: '',
            AmazonS3Region: '',
            AmazonS3Endpoint: '',
            AmazonS3AccessKeyId: '',
            AmazonS3SecretAccessKey: '',
            AmazonS3SSL: false,
            AmazonS3SSE: false,
            ...initialValue?.Settings,
        },
    });

    return useMemo(() => ({
        formState,
        setEnable: (value: boolean) => {
            const newState = {
                ...formState,
                Enable: value,
            };
            setFormState(newState);
            onChange('PluginSettings.Plugins.com+mattermost+plugin-legal-hold.amazons3bucketsettings', newState);
        },
        setFormValue: <T extends keyof FileSettings>(key: T, value: FileSettings[T]) => {
            const newState = {
                ...formState,
                Settings: {
                    ...formState.Settings,
                    [key]: value,
                },
            };

            setFormState(newState);
            onChange('PluginSettings.Plugins.com+mattermost+plugin-legal-hold.amazons3bucketsettings', newState);
        },
    }), [formState, setFormState, onChange]);
};

type Props = {
    value: AmazonS3BucketSettingsData | undefined;
    onChange: (id: string, value: AmazonS3BucketSettingsData) => void;
};

const isSettingFormDirty = () => {
    const submitButton = document.querySelector('button#saveSetting') as HTMLButtonElement | null;
    if (submitButton) {
        return !submitButton.disabled;
    }

    return false;
};

const AmazonS3BucketSettings = (props: Props) => {
    const {formState, setFormValue, setEnable} = useS3BucketForm(props.value, props.onChange);
    const [testingConnection, setTestingConnection] = useState(false);

    const [message, setMessage] = useState('');
    const [error, setError] = useState('');

    const s3Settings = formState.Settings;

    const testConnection = async () => {
        if (isSettingFormDirty()) {
            setError('Please save the settings before testing the connection.');
            return;
        }

        setMessage('');
        setError('');

        setTestingConnection(true);

        try {
            const res = await Client.testAmazonS3Connection();
            if (res.message) {
                setMessage(res.message);
            }
        } catch (err) {
            if ('message' in (err as Error)) {
                setError((err as Error).message);
            }
        }

        setTestingConnection(false);
    };

    let statusMessage: React.ReactNode | undefined;
    if (error) {
        statusMessage = (
            <StatusMessage
                state='warn'
                message={error}
            />
        );
    } else if (message) {
        statusMessage = (
            <StatusMessage
                state='success'
                message={message}
            />
        );
    }

    return (
        <IntlProvider locale='en-US'>
            <BooleanSetting
                id='com.mattermost.plugin-legal-hold.EnableCustomS3Bucket'
                name='Enable Custom S3 Bucket'
                helpText='When enabled, the plugin will use the custom S3 bucket settings.'
                value={formState.Enable}
                onChange={(value) => setEnable(value)}
            />
            <TextSetting
                id='com.mattermost.plugin-legal-hold.AmazonS3Bucket'
                name='Amazon S3 Bucket'
                helpText='The name of the Amazon S3 bucket to store the files.'
                value={s3Settings.AmazonS3Bucket}
                onChange={(value) => setFormValue('AmazonS3Bucket', value)}
                disabled={!formState.Enable}
            />
            <TextSetting
                id='com.mattermost.plugin-legal-hold.AmazonS3PathPrefix'
                name='Amazon S3 Path Prefix'
                helpText='The path prefix to store the files in the Amazon S3 bucket.'
                value={s3Settings.AmazonS3PathPrefix}
                onChange={(value) => setFormValue('AmazonS3PathPrefix', value)}
                disabled={!formState.Enable}
            />
            <TextSetting
                id='com.mattermost.plugin-legal-hold.AmazonS3Region'
                name='Amazon S3 Region'
                helpText='The region of the Amazon S3 bucket.'
                value={s3Settings.AmazonS3Region}
                onChange={(value) => setFormValue('AmazonS3Region', value)}
                disabled={!formState.Enable}
            />
            <TextSetting
                id='com.mattermost.plugin-legal-hold.AmazonS3AccessKeyId'
                name='Amazon S3 Access Key ID'
                helpText='The access key ID to access the Amazon S3 bucket.'
                value={s3Settings.AmazonS3AccessKeyId}
                onChange={(value) => setFormValue('AmazonS3AccessKeyId', value)}
                disabled={!formState.Enable}
            />
            <TextSetting
                id='com.mattermost.plugin-legal-hold.AmazonS3Endpoint'
                name='Amazon S3 Endpoint'
                helpText='The endpoint of the Amazon S3 bucket.'
                value={s3Settings.AmazonS3Endpoint}
                onChange={(value) => setFormValue('AmazonS3Endpoint', value)}
                disabled={!formState.Enable}
            />
            <TextSetting
                id='com.mattermost.plugin-legal-hold.AmazonS3SecretAccessKey'
                name='Amazon S3 Secret Access Key'
                helpText='The secret access key to access the Amazon S3 bucket.'
                value={s3Settings.AmazonS3SecretAccessKey}
                onChange={(value) => setFormValue('AmazonS3SecretAccessKey', value)}
                disabled={!formState.Enable}
            />
            <BooleanSetting
                id='com.mattermost.plugin-legal-hold.AmazonS3SSL'
                name='Amazon S3 SSL'
                helpText='When enabled, the connection to the Amazon S3 bucket will be encrypted.'
                value={s3Settings.AmazonS3SSL}
                onChange={(value) => setFormValue('AmazonS3SSL', value)}
                disabled={!formState.Enable}
            />
            <BooleanSetting
                id='com.mattermost.plugin-legal-hold.AmazonS3SSE'
                name='Amazon S3 SSE'
                helpText='When enabled, the server-side encryption will be enabled for the Amazon S3 bucket.'
                value={s3Settings.AmazonS3SSE}
                onChange={(value) => setFormValue('AmazonS3SSE', value)}
                disabled={!formState.Enable}
            />
            <BaseSetting
                helpText=''
                id=''
                name=''
            >
                <SaveButton
                    type='button'
                    btnClass='btn-tertiary'
                    saving={testingConnection}
                    savingMessage={''}
                    defaultMessage={'Test Connection'}
                    onClick={testConnection}
                    disabled={!formState.Enable}
                />
                {statusMessage}
            </BaseSetting>
        </IntlProvider>
    );
};

export default AmazonS3BucketSettings;
