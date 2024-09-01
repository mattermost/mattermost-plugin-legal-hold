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
    runLegalHold: Function;
}

const LegalHoldRow = (props: LegalHoldRowProps) => {
    const lh = props.legalHold;
    const startsAt = (new Date(lh.starts_at)).toLocaleDateString();
    const endsAt = lh.ends_at === 0 ? 'Never' : (new Date(lh.ends_at)).toLocaleDateString();

    const release = () => {
        props.releaseLegalHold(lh);
    };

    const run = () => {
        props.runLegalHold(lh);
    }

    const usernames = props.users.map((user) => {
        if (user) {
            return `@${user.username} `;
        }
        return 'loading...';
    });

    const downloadUrl = Client.downloadUrl(lh.id);

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
                            marginRight: '20px',
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
                <a
                    href='#'
                    onClick={release}
                    className={'btn btn-danger'}
                >{'Release'}</a>
                <a
                    href='#'
                    onClick={run}
                    className={'btn btn-info'}
                >{'Run Now'}</a>
            </div>
        </React.Fragment>
    );
};

export default LegalHoldRow;
