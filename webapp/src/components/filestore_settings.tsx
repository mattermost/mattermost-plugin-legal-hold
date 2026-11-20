import React, {useMemo, useState} from 'react';

import {IntlProvider} from 'react-intl';

import Client from '@/client';

import BooleanSetting from './admin_console_settings/boolean_setting';
import TextSetting from './admin_console_settings/text_setting';
import SaveButton from './mattermost-webapp/save_button';
import BaseSetting from './admin_console_settings/base_setting';
import StatusMessage from './admin_console_settings/status_message';
import SecretTextSetting from './admin_console_settings/secret_text_setting';

type DriverName = 'server' | 'amazons3' | 'azureblob' | 'local';

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

type AzureSettings = {
    StorageAccount: string;
    AuthMode: string;
    AccessKey: string;
    Container: string;
    PathPrefix: string;
    Cloud: string;
    Endpoint: string;
    SSL: boolean;
    RequestTimeoutMilliseconds: number;
};

type FilestoreSettingsData = {
    Enable: boolean;
    DriverName: DriverName;
    Settings: FileSettings;
    Azure: AzureSettings;
    Directory: string;
};

const CONFIG_ID = 'PluginSettings.Plugins.com+mattermost+plugin-legal-hold.amazons3bucketsettings';

const defaultS3 = (): FileSettings => ({
    DriverName: 'amazons3',
    AmazonS3RequestTimeoutMilliseconds: 30000,
    AmazonS3Bucket: '',
    AmazonS3PathPrefix: '',
    AmazonS3Region: '',
    AmazonS3Endpoint: '',
    AmazonS3AccessKeyId: '',
    AmazonS3SecretAccessKey: '',
    AmazonS3SSL: true,
    AmazonS3SSE: false,
});

const defaultAzure = (): AzureSettings => ({
    StorageAccount: '',
    AuthMode: 'shared_key',
    AccessKey: '',
    Container: '',
    PathPrefix: '',
    Cloud: 'commercial',
    Endpoint: '',
    SSL: true,
    RequestTimeoutMilliseconds: 30000,
});

// A config saved before DriverName existed used Enable to mean "dedicated S3".
const initialDriver = (value: FilestoreSettingsData | undefined): DriverName => {
    if (value?.DriverName === 'server' || value?.DriverName === 'amazons3' || value?.DriverName === 'azureblob' || value?.DriverName === 'local') {
        return value.DriverName;
    }
    if (value?.Enable) {
        return 'amazons3';
    }
    return 'server';
};

const useFilestoreForm = (initialValue: FilestoreSettingsData | undefined, onChange: (id: string, value: FilestoreSettingsData) => void) => {
    const driver = initialDriver(initialValue);
    const [formState, setFormState] = useState<FilestoreSettingsData>({
        Enable: driver === 'amazons3',
        DriverName: driver,
        Settings: {
            ...defaultS3(),
            ...initialValue?.Settings,
            DriverName: 'amazons3',
        },
        Azure: {
            ...defaultAzure(),
            ...initialValue?.Azure,
        },
        Directory: initialValue?.Directory ?? '',
    });

    const publish = (next: FilestoreSettingsData) => {
        setFormState(next);
        onChange(CONFIG_ID, next);
    };

    return useMemo(() => ({
        formState,
        setDriver: (value: DriverName) => {
            publish({
                ...formState,
                DriverName: value,
                Enable: value === 'amazons3',
            });
        },
        setS3: <T extends keyof FileSettings>(key: T, value: FileSettings[T]) => {
            publish({
                ...formState,
                Settings: {
                    ...formState.Settings,
                    [key]: value,
                },
            });
        },
        setAzure: <T extends keyof AzureSettings>(key: T, value: AzureSettings[T]) => {
            publish({
                ...formState,
                Azure: {
                    ...formState.Azure,
                    [key]: value,
                },
            });
        },
        setDirectory: (value: string) => {
            publish({
                ...formState,
                Directory: value,
            });
        },
    }), [formState, onChange]); // eslint-disable-line react-hooks/exhaustive-deps
};

type Props = {
    value: FilestoreSettingsData | undefined;
    onChange: (id: string, value: FilestoreSettingsData) => void;
};

const isSettingFormDirty = () => {
    const submitButton = document.querySelector('button#saveSetting') as HTMLButtonElement | null;
    if (submitButton) {
        return !submitButton.disabled;
    }
    return false;
};

const driverHelp: Record<DriverName, string> = {
    server: 'Legal holds are written to the file store configured under Environment > File Storage. That covers local disk, a blobfuse mount, Amazon S3, and Azure Blob.',
    amazons3: 'Legal holds are written to this S3 bucket instead of the Mattermost file store.',
    azureblob: 'Legal holds are written to this Azure Blob container instead of the Mattermost file store.',
    local: 'Legal holds are written to this directory on the Mattermost server. Use a path on a blobfuse mount when the container is mounted as a filesystem.',
};

const FilestoreSettings = (props: Props) => {
    const {formState, setDriver, setS3, setAzure, setDirectory} = useFilestoreForm(props.value, props.onChange);
    const [testingConnection, setTestingConnection] = useState(false);
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');

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

    const s3 = formState.Settings;
    const azure = formState.Azure;

    return (
        <IntlProvider locale='en-US'>
            <BaseSetting
                id='com.mattermost.plugin-legal-hold.FilestoreDriver'
                name='Filestore'
                helpText={driverHelp[formState.DriverName]}
            >
                <select
                    id='com.mattermost.plugin-legal-hold.FilestoreDriver'
                    className='form-control'
                    value={formState.DriverName}
                    onChange={(e) => setDriver(e.target.value as DriverName)}
                >
                    <option value='server'>{'Mattermost file storage'}</option>
                    <option value='amazons3'>{'Amazon S3'}</option>
                    <option value='azureblob'>{'Azure Blob'}</option>
                    <option value='local'>{'Local directory'}</option>
                </select>
            </BaseSetting>
            {formState.DriverName === 'amazons3' && (
                <>
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3Bucket'
                        name='Amazon S3 Bucket'
                        helpText='The name of the Amazon S3 bucket to store the files.'
                        value={s3.AmazonS3Bucket}
                        onChange={(value) => setS3('AmazonS3Bucket', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3PathPrefix'
                        name='Amazon S3 Path Prefix'
                        helpText='The path prefix to store the files in the Amazon S3 bucket.'
                        value={s3.AmazonS3PathPrefix}
                        onChange={(value) => setS3('AmazonS3PathPrefix', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3Region'
                        name='Amazon S3 Region'
                        helpText='The region of the Amazon S3 bucket.'
                        value={s3.AmazonS3Region}
                        onChange={(value) => setS3('AmazonS3Region', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3AccessKeyId'
                        name='Amazon S3 Access Key ID'
                        helpText='The access key ID to access the Amazon S3 bucket.'
                        value={s3.AmazonS3AccessKeyId}
                        onChange={(value) => setS3('AmazonS3AccessKeyId', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3Endpoint'
                        name='Amazon S3 Endpoint'
                        helpText='The endpoint of the Amazon S3 bucket.'
                        value={s3.AmazonS3Endpoint}
                        onChange={(value) => setS3('AmazonS3Endpoint', value)}
                    />
                    <SecretTextSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3SecretAccessKey'
                        name='Amazon S3 Secret Access Key'
                        helpText='The secret access key to access the Amazon S3 bucket.'
                        value={s3.AmazonS3SecretAccessKey}
                        onChange={(value) => setS3('AmazonS3SecretAccessKey', value)}
                    />
                    <BooleanSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3SSL'
                        name='Amazon S3 SSL'
                        helpText='When enabled, the connection to the Amazon S3 bucket will be encrypted.'
                        value={s3.AmazonS3SSL}
                        onChange={(value) => setS3('AmazonS3SSL', value)}
                    />
                    <BooleanSetting
                        id='com.mattermost.plugin-legal-hold.AmazonS3SSE'
                        name='Amazon S3 SSE'
                        helpText='When enabled, the server-side encryption will be enabled for the Amazon S3 bucket.'
                        value={s3.AmazonS3SSE}
                        onChange={(value) => setS3('AmazonS3SSE', value)}
                    />
                </>
            )}
            {formState.DriverName === 'azureblob' && (
                <>
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AzureStorageAccount'
                        name='Storage account'
                        helpText='The Azure storage account name.'
                        value={azure.StorageAccount}
                        onChange={(value) => setAzure('StorageAccount', value)}
                    />
                    <BaseSetting
                        id='com.mattermost.plugin-legal-hold.AzureAuthMode'
                        name='Authentication'
                        helpText='Shared key uses the account access key. Default credential uses the pod identity (managed identity, workload identity, or service principal environment variables).'
                    >
                        <select
                            id='com.mattermost.plugin-legal-hold.AzureAuthMode'
                            className='form-control'
                            value={azure.AuthMode}
                            onChange={(e) => setAzure('AuthMode', e.target.value)}
                        >
                            <option value='shared_key'>{'Shared key'}</option>
                            <option value='default_credential'>{'Default credential'}</option>
                        </select>
                    </BaseSetting>
                    {azure.AuthMode !== 'default_credential' && (
                        <SecretTextSetting
                            id='com.mattermost.plugin-legal-hold.AzureAccessKey'
                            name='Access key'
                            helpText='The storage account access key. Leave this empty when using default credential.'
                            value={azure.AccessKey}
                            onChange={(value) => setAzure('AccessKey', value)}
                        />
                    )}
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AzureContainer'
                        name='Container'
                        helpText='The blob container that will hold legal hold data.'
                        value={azure.Container}
                        onChange={(value) => setAzure('Container', value)}
                    />
                    <TextSetting
                        id='com.mattermost.plugin-legal-hold.AzurePathPrefix'
                        name='Path prefix'
                        helpText='Optional prefix inside the container. Legal hold files are stored under this prefix.'
                        value={azure.PathPrefix}
                        onChange={(value) => setAzure('PathPrefix', value)}
                    />
                    <BaseSetting
                        id='com.mattermost.plugin-legal-hold.AzureCloud'
                        name='Cloud'
                        helpText='Commercial and Government use the public Azure endpoints. Custom takes the endpoint you enter, for a proxy or Azurite.'
                    >
                        <select
                            id='com.mattermost.plugin-legal-hold.AzureCloud'
                            className='form-control'
                            value={azure.Cloud}
                            onChange={(e) => setAzure('Cloud', e.target.value)}
                        >
                            <option value='commercial'>{'Commercial'}</option>
                            <option value='government'>{'Government'}</option>
                            <option value='custom'>{'Custom endpoint'}</option>
                        </select>
                    </BaseSetting>
                    {azure.Cloud === 'custom' && (
                        <TextSetting
                            id='com.mattermost.plugin-legal-hold.AzureEndpoint'
                            name='Endpoint'
                            helpText='Full blob service URL, including the scheme and account. Example: https://account.blob.core.windows.net/'
                            value={azure.Endpoint}
                            onChange={(value) => setAzure('Endpoint', value)}
                        />
                    )}
                    <BooleanSetting
                        id='com.mattermost.plugin-legal-hold.AzureSSL'
                        name='SSL'
                        helpText='Use HTTPS. Turn this off only for a local emulator.'
                        value={azure.SSL}
                        onChange={(value) => setAzure('SSL', value)}
                    />
                </>
            )}
            {formState.DriverName === 'local' && (
                <TextSetting
                    id='com.mattermost.plugin-legal-hold.Directory'
                    name='Directory'
                    helpText='Absolute path the Mattermost process can write. For blobfuse, this is the mount path, not an account name.'
                    value={formState.Directory}
                    onChange={setDirectory}
                />
            )}
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
                />
                {statusMessage}
            </BaseSetting>
        </IntlProvider>
    );
};

export default FilestoreSettings;
