import React, {useEffect, useMemo, useState} from 'react';

import {IntlProvider} from 'react-intl';

import BooleanSetting from './admin_console_settings/boolean_setting';
import TextSetting from './admin_console_settings/text_setting';

type AmazonS3BucketSettingsData = {
    EnableS3Settings: boolean;
    AmazonS3Bucket: string;
    AmazonS3PathPrefix: string;
    AmazonS3Region: string;
    AmazonS3AccessKeyId: string;
    AmazonS3Endpoint: string;
    AmazonS3SecretAccessKey: string;
    AmazonS3SSL: boolean;
    AmazonS3SSE: boolean;
};

const useS3BucketForm = (initialValue: AmazonS3BucketSettingsData | undefined, onChange: (id: string, value: AmazonS3BucketSettingsData) => void) => {
    const [formState, setFormState] = useState<AmazonS3BucketSettingsData>(initialValue || {
        EnableS3Settings: false,
        AmazonS3Bucket: '',
        AmazonS3PathPrefix: '',
        AmazonS3Region: '',
        AmazonS3AccessKeyId: '',
        AmazonS3Endpoint: '',
        AmazonS3SecretAccessKey: '',
        AmazonS3SSL: false,
        AmazonS3SSE: false,
    });

    return useMemo(() => ({
        formState,
        setFormValue: <T extends keyof AmazonS3BucketSettingsData>(key: T, value: AmazonS3BucketSettingsData[T]) => {
            setFormState((prev) => ({
                ...prev,
                [key]: value,
            }));
            onChange('PluginSettings.Plugins.com+mattermost+plugin-legal-hold.s3bucketsettings', {
                ...formState,
                [key]: value,
            });
        },
    }), [formState, setFormState]);
};

type Props = {
    value: AmazonS3BucketSettingsData | undefined;
    onChange: (id: string, value: AmazonS3BucketSettingsData) => void;
};

const S3BucketSettings = (props: Props) => {
    const {formState, setFormValue} = useS3BucketForm(props.value, props.onChange);

    const testConnection = async () => {
        // alert(`Testing connection ${JSON.stringify(formState)}`);
    };

    const [showS3Settings, setShowS3Settings] = useState(true);

    return (
        <IntlProvider locale='en-US'>
            <BooleanSetting
                id='com.mattermost.plugin-legal-hold.EnableCustomS3Bucket'
                name='Enable Custom S3 Bucket'
                helpText='When enabled, the plugin will use the custom S3 bucket settings.'
                value={formState.EnableS3Settings}
                onChange={(value) => setFormValue('EnableS3Settings', value)}
            />
            {formState.EnableS3Settings && (
                <>
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3Bucket'
                        name='Amazon S3 Bucket'
                        helpText='The name of the Amazon S3 bucket to store the files.'
                        value={formState.AmazonS3Bucket}
                        onChange={(value) => setFormValue('AmazonS3Bucket', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3PathPrefix'
                        name='Amazon S3 Path Prefix'
                        helpText='The path prefix to store the files in the Amazon S3 bucket.'
                        value={formState.AmazonS3PathPrefix}
                        onChange={(value) => setFormValue('AmazonS3PathPrefix', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3Region'
                        name='Amazon S3 Region'
                        helpText='The region of the Amazon S3 bucket.'
                        value={formState.AmazonS3Region}
                        onChange={(value) => setFormValue('AmazonS3Region', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3AccessKeyId'
                        name='Amazon S3 Access Key ID'
                        helpText='The access key ID to access the Amazon S3 bucket.'
                        value={formState.AmazonS3AccessKeyId}
                        onChange={(value) => setFormValue('AmazonS3AccessKeyId', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3Endpoint'
                        name='Amazon S3 Endpoint'
                        helpText='The endpoint of the Amazon S3 bucket.'
                        value={formState.AmazonS3Endpoint}
                        onChange={(value) => setFormValue('AmazonS3Endpoint', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3SecretAccessKey'
                        name='Amazon S3 Secret Access Key'
                        helpText='The secret access key to access the Amazon S3 bucket.'
                        value={formState.AmazonS3SecretAccessKey}
                        onChange={(value) => setFormValue('AmazonS3SecretAccessKey', value)}
                    />
                    <BooleanSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3SSL'
                        name='Amazon S3 SSL'
                        helpText='When enabled, the connection to the Amazon S3 bucket will be encrypted.'
                        value={formState.AmazonS3SSL}
                        onChange={(value) => setFormValue('AmazonS3SSL', value)}
                    />
                    <BooleanSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3SSE'
                        name='Amazon S3 SSE'
                        helpText='When enabled, the server-side encryption will be enabled for the Amazon S3 bucket.'
                        value={formState.AmazonS3SSE}
                        onChange={(value) => setFormValue('AmazonS3SSE', value)}
                    />
                    <button
                        type='button'
                        onClick={testConnection}
                    >
                        {'Test Connection'}
                    </button>
                </>
            )}
        </IntlProvider>
    );
};

export default S3BucketSettings;
