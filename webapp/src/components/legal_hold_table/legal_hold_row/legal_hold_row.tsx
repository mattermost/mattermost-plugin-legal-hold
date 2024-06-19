import React from 'react';
import {UserProfile} from 'mattermost-redux/types/users';

import {LegalHold} from '@/types';
import Client from '@/client';

import Tooltip from '@/components/mattermost-webapp/tooltip';

import OverlayTrigger from '@/components/mattermost-webapp/overlay_trigger';

import DownloadIcon from './download-outline_F0B8F.svg';
import EditIcon from './pencil-outline_F0CB6.svg';

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

    const usernames = props.users.map((user) => {
        if (user) {
            return `@${user.username} `;
        }
        return 'loading...';
    });

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

    const bundleUrl = Client.bundleUrl(lh.id);
    const bundleButton = (
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
            <a
                href={bundleUrl}
                style={{
                    marginRight: '10px',
                    height: '24px',
                    transform: 'rotate(180deg)',
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
                    transform: 'rotate(180deg)',
                    fill: 'rgba(0, 0, 0, 0.2)',
                    cursor: 'not-allowed',
                }}
            >
                <DownloadIcon/>
            </span>
        </OverlayTrigger>
    );

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
                {(lh.locks?.includes('bundle')) ? disabledBundleButton : bundleButton}
                <a
                    href='#'
                    onClick={release}
                    className={'btn btn-danger'}
                >{'Release'}</a>
            </div>
        </React.Fragment>
    );
};

export default LegalHoldRow;
