import {useState} from 'react';

import DownloadOutlineIcon from '@mattermost/compass-icons/components/download-outline';
import EyeOutlineIcon from '@mattermost/compass-icons/components/eye-outline';
import PencilOutlineIcon from '@mattermost/compass-icons/components/pencil-outline';
import PlayOutlineIcon from '@mattermost/compass-icons/components/play-outline';
import {Button} from '@mattermost/compass-ui/components/button';
import {Icon} from '@mattermost/compass-ui/components/icon';
import {IconButton} from '@mattermost/compass-ui/components/icon-button';

import RunConfirmationModal from './run_confirmation_modal';
import RunErrorModal from './run_error_modal';

import Client from '@/client';
import type {LegalHold} from '@/types';

interface LegalHoldRowProps {
    legalHold: LegalHold;
    releaseLegalHold: (legalHold: LegalHold) => void;
    showUpdateModal: (legalHold: LegalHold) => void;
    showSecretModal: (legalHold: LegalHold) => void;
    runLegalHold: (id: string) => Promise<unknown>;
    refresh: () => void;
}

// The forms store a date as midnight UTC, so the table shows the date in UTC as well.
const formatDate = (timestamp: number) => new Date(timestamp).toLocaleDateString(undefined, {timeZone: 'UTC'});

const getLastRunDisplay = (lh: LegalHold) => {
    if (lh.status === 'executing') {
        return 'Running now...';
    }
    if (!lh.last_execution_ended_at || lh.last_execution_ended_at === 0) {
        return 'Never';
    }

    return new Date(lh.last_execution_ended_at).toLocaleString();
};

const LegalHoldRow = (props: LegalHoldRowProps) => {
    const [showRunConfirmModal, setShowRunConfirmModal] = useState(false);
    const [showRunErrorModal, setShowRunErrorModal] = useState(false);
    const lh = props.legalHold;
    const startsAt = formatDate(lh.starts_at);
    const endsAt = lh.ends_at === 0 ? 'Never' : formatDate(lh.ends_at);
    const isExecuting = lh.status === 'executing';
    const canDownload = !isExecuting && lh.has_messages;

    const userCount = lh.user_ids?.length ?? 0;
    const groupCount = lh.group_ids?.length ?? 0;

    const label = (action: string) => `${lh.display_name} ${action} button`;

    // compass-gap: Compass Tooltip has no hover wiring (compass-design#82), so the icon buttons use a
    // native title until it does.
    return (
        <div
            role='row'
            className='legal-hold-table__row'
        >
            <div
                role='cell'
                data-testid={`name-${lh.id}`}
                data-legalholdid={lh.id}
            >{lh.display_name}</div>
            <div
                role='cell'
                data-testid={`start-date-${lh.id}`}
            >{startsAt}</div>
            <div
                role='cell'
                data-testid={`end-date-${lh.id}`}
            >{endsAt}</div>
            <div
                role='cell'
                data-testid={`users-${lh.id}`}
            >{userCount} {'users'}</div>
            <div
                role='cell'
                data-testid={`groups-${lh.id}`}
            >{groupCount} {'groups'}</div>
            <div
                role='cell'
                data-testid={`last-run-${lh.id}`}
            >
                {getLastRunDisplay(lh)}
            </div>
            <div
                role='cell'
                className='legal-hold-table__actions'
            >
                <IconButton
                    data-testid={`update-${lh.id}`}
                    aria-label={label('update')}
                    title='Update legal hold'
                    size='small'
                    icon={<Icon glyph={<PencilOutlineIcon/>}/>}
                    onClick={() => props.showUpdateModal(lh)}
                />
                <IconButton
                    data-testid={`show-${lh.id}`}
                    aria-label={label('show secret')}
                    title='Show legal hold secret'
                    size='small'
                    icon={<Icon glyph={<EyeOutlineIcon/>}/>}
                    onClick={() => props.showSecretModal(lh)}
                />
                <IconButton
                    asChild={true}
                    aria-label={label('download')}
                    size='small'
                    disabled={!canDownload}
                    icon={<Icon glyph={<DownloadOutlineIcon/>}/>}
                >
                    <a
                        data-testid={`download-${lh.id}`}
                        title='Download legal hold'
                        href={canDownload ? Client.downloadUrl(lh.id) : undefined}
                        download={canDownload}
                    />
                </IconButton>
                <IconButton
                    data-testid={`run-${lh.id}`}
                    aria-label={label('run')}
                    title='Run legal hold now'
                    size='small'
                    disabled={isExecuting}
                    icon={<Icon glyph={<PlayOutlineIcon/>}/>}
                    onClick={() => setShowRunConfirmModal(true)}
                />
                <Button
                    data-testid={`release-${lh.id}`}
                    aria-label={label('release')}
                    destructive={true}
                    emphasis='secondary'
                    size='small'
                    disabled={isExecuting}
                    onClick={() => props.releaseLegalHold(lh)}
                >
                    {'Release'}
                </Button>
            </div>
            <RunConfirmationModal
                show={showRunConfirmModal}
                onHide={() => setShowRunConfirmModal(false)}
                onConfirm={() => {
                    setShowRunConfirmModal(false);
                    props.runLegalHold(lh.id).catch(() => {
                        setShowRunErrorModal(true);
                    }).finally(props.refresh);
                }}
            />
            <RunErrorModal
                show={showRunErrorModal}
                onHide={() => setShowRunErrorModal(false)}
            />
        </div>
    );
};

export default LegalHoldRow;
