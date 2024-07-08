import React, {useState} from 'react';
import {UserProfile} from 'mattermost-redux/types/users';

import {LegalHold} from '@/types';
import Client from '@/client';

import Tooltip from '@/components/mattermost-webapp/tooltip';

import OverlayTrigger from '@/components/mattermost-webapp/overlay_trigger';
import StatusMessage from '@/components/admin_console_settings/status_message';

import DownloadIcon from './download-outline_F0B8F.svg';
import UploadIcon from './upload-outline_F0E07.svg';
import EditIcon from './pencil-outline_F0CB6.svg';
import LoadingIcon from './loading.svg';

interface LegalHoldRowProps {
    legalHold: LegalHold;
    users: UserProfile[];
    releaseLegalHold: Function;
    showUpdateModal: Function;
}

const LegalHoldRow = (props: LegalHoldRowProps) => {
    const lh = props.legalHold;
    const startsAt = (new Date(lh.starts_at)).toLocaleDateString();
    const endsAt = lh.ends_at === 0 ? 'Never' : (new Date(lh.ends_at)).toLocaleDateString();

    const release = () => {
        props.releaseLegalHold(lh);
    };

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');

    const bundleLegalHold = async (legalHold: LegalHold) => {
        try {
            await Client.bundleLegalHold(legalHold.id);
            lh.locks?.push('bundle');
        } catch (err) {
            if ('message' in (err as Error)) {
                setError((err as Error).message);
            }
        }
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

    const downloadUrl = Client.downloadUrl(lh.id);

    const downloadButton = (
        <OverlayTrigger

            // @ts-ignore
            delayShow={300}
            placement='top'
            overlay={(
                <Tooltip id={'DownloadLegalHoldTooltip'}>
                    {'Download Legal Hold'}
                </Tooltip>
            )}
        >
            <a
                href={downloadUrl}
                style={{
                    marginRight: '10px',
                    height: '24px',
                }}
            >
                <span
                    style={{
                        fill: 'rgba(0, 0, 0, 0.5)',
                    }}
                >
                    <DownloadIcon/>
                </span>
            </a>
        </OverlayTrigger>
    );

    const enabledBundleButton = (
        <OverlayTrigger

            // @ts-ignore
            delayShow={300}
            placement='top'
            overlay={(
                <Tooltip id={'BundleLegalHoldTooltip'}>
                    {'Upload Legal Hold in file store'}
                </Tooltip>
            )}
        >
            <span
                onClick={() => bundleLegalHold(lh)}
                style={{
                    marginRight: '10px',
                    height: '24px',
                    fill: 'rgba(0, 0, 0, 0.5)',
                    cursor: 'pointer',
                }}
            >
                <UploadIcon/>
            </span>
        </OverlayTrigger>
    );

    const disabledBundleButton = (
        <OverlayTrigger

            // @ts-ignore
            delayShow={300}
            placement='top'
            overlay={(
                <Tooltip id={'BundleLegalHoldTooltip'}>
                    {'Can\'t upload Legal Hold because Another job is running'}
                </Tooltip>
            )}
        >
            <span
                style={{
                    marginRight: '10px',
                    height: '24px',
                    width: '24px',
                    fill: 'rgba(0, 0, 0, 0.5)',
                    cursor: 'not-allowed',
                }}
            >
                <LoadingIcon
                    style={{
                        animation: 'spin 2s linear infinite',
                    }}
                />
            </span>
        </OverlayTrigger>
    );

    const bundleButton = (lh.locks?.includes('bundle')) ? disabledBundleButton : enabledBundleButton;

    return (
        <React.Fragment>
            <div>{lh.display_name}</div>
            <div>{startsAt}</div>
            <div>{endsAt}</div>
            <div>{props.users.length} {'users'}</div>
            <div
                style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                }}
            >
                <OverlayTrigger

                    // @ts-ignore
                    delayShow={300}
                    placement='top'
                    overlay={(
                        <Tooltip id={'UpdateLegalHoldTooltip'}>
                            {'Update Legal Hold'}
                        </Tooltip>
                    )}
                >
                    <a
                        href='#'
                        onClick={() => props.showUpdateModal(lh)}
                        style={{
                            marginRight: '10px',
                            height: '24px',
                        }}
                    >
                        <span
                            style={{
                                fill: 'rgba(0, 0, 0, 0.5)',
                            }}
                        >
                            <EditIcon/>
                        </span>
                    </a>
                </OverlayTrigger>
                {downloadButton}
                {/* {bundleButton} */}
                <a
                    href='#'
                    onClick={release}
                    className={'btn btn-danger'}
                >{'Release'}</a>
            </div>
            {(statusMessage) ? (<React.Fragment>{statusMessage}</React.Fragment>) : null}
        </React.Fragment>
    );
};

export default LegalHoldRow;
