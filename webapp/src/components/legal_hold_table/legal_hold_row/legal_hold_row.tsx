import React, {useState} from 'react';
import {UserProfile} from 'mattermost-redux/types/users';

import {LegalHold} from '@/types';

import Tooltip from '@/components/mattermost-webapp/tooltip';

import OverlayTrigger from '@/components/mattermost-webapp/overlay_trigger';

import DownloadIcon from './download-outline_F0B8F.svg';
import EditIcon from './pencil-outline_F0CB6.svg';
import EyeLockIcon from './eye-outline_F06D0.svg';
import RunIcon from './play-outline.svg';
import RunConfirmationModal from './run_confirmation_modal';
import RunErrorModal from './run_error_modal';

interface LegalHoldRowProps {
    legalHold: LegalHold;
    users: UserProfile[];
    releaseLegalHold: Function;
    showUpdateModal: Function;
    showSecretModal: Function;
    runLegalHold: (id: string) => Promise<void>;
}

const LegalHoldRow = (props: LegalHoldRowProps) => {
    const [showRunConfirmModal, setShowRunConfirmModal] = useState(false);
    const [showRunErrorModal, setShowRunErrorModal] = useState(false);
    const lh = props.legalHold;
    const startsAt = (new Date(lh.starts_at)).toLocaleDateString();
    const endsAt = lh.ends_at === 0 ? 'Never' : (new Date(lh.ends_at)).toLocaleDateString();

    const release = () => {
        props.releaseLegalHold(lh);
    };

    const downloadUrl = `/plugins/com.mattermost.plugin-legal-hold/api/v1/legalholds/${lh.id}/download`;

    return (
        <React.Fragment>
            <div
                data-testid={`name-${lh.id}`}
                data-legalholdid={lh.id}
            >{lh.display_name}</div>
            <div data-testid={`start-date-${lh.id}`}>{startsAt}</div>
            <div data-testid={`end-date-${lh.id}`}>{endsAt}</div>
            <div data-testid={`users-${lh.id}`}>{props.users.length} {'users'}</div>
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
                            {'Update legal hold'}
                        </Tooltip>
                    )}
                >
                    <a
                        data-testid={`update-${lh.id}`}
                        aria-label={`${lh.display_name} update button`}
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
                        <Tooltip id={'ShowLegalHoldSecret'}>
                            {'Show Legal Hold Secret'}
                        </Tooltip>
                    )}
                >
                    <a
                        data-testid={`show-${lh.id}`}
                        aria-label={`${lh.display_name} show secret button`}
                        href='#'
                        onClick={() => props.showSecretModal(lh)}
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
                            <EyeLockIcon/>
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
                        data-testid={`download-${lh.id}`}
                        aria-label={`${lh.display_name} download button`}
                        href={downloadUrl}
                        download={true}
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
                <OverlayTrigger

                    // @ts-ignore
                    delayShow={300}
                    placement='top'
                    overlay={(
                        <Tooltip id={'RunLegalHoldTooltip'}>
                            {'Run Legal Hold Now'}
                        </Tooltip>
                    )}
                >
                    <a
                        data-testid={`run-${lh.id}`}
                        aria-label={`${lh.display_name} run button`}
                        href='#'
                        onClick={() => setShowRunConfirmModal(true)}
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
                            <RunIcon/>
                        </span>
                    </a>
                </OverlayTrigger>
                <a
                    data-testid={`release-${lh.id}`}
                    role='button'
                    aria-label={`${lh.display_name} release button`}
                    href='#'
                    onClick={release}
                    className={'btn btn-danger'}
                >{'Release'}</a>
            </div>
            <RunConfirmationModal
                show={showRunConfirmModal}
                onHide={() => setShowRunConfirmModal(false)}
                onConfirm={() => {
                    setShowRunConfirmModal(false);
                    props.runLegalHold(lh.id).then(() => {
                        // Success - do nothing as the message from the API is enough
                    }).catch(() => {
                        setShowRunErrorModal(true);
                    });
                }}
            />
            <RunErrorModal
                show={showRunErrorModal}
                onHide={() => setShowRunErrorModal(false)}
            />
        </React.Fragment>
    );
};

export default LegalHoldRow;
