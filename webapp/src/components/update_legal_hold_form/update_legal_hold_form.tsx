import {useEffect, useState} from 'react';
import type {FormEvent} from 'react';
import {useDispatch} from 'react-redux';

import {Button} from '@mattermost/compass-ui/components/button';
import {Modal, ModalOverlay} from '@mattermost/compass-ui/components/modal';
import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';
import type {Group} from '@mattermost/types/groups';
import type {UserProfile} from '@mattermost/types/users';

import {getMissingProfilesByIds} from 'mattermost-redux/actions/users';

import {getMissingGroupsByIds} from '@/actions/groups';
import {useLegalHoldEntities} from '@/components/legal_hold_entities';
import LegalHoldFormFields from '@/components/legal_hold_form_fields';
import type {LegalHold, UpdateLegalHold} from '@/types';

import '../legal_hold_form.css';

const FORM_ID = 'edit-legal-hold-form';

// The forms store a date as midnight UTC, so a stored date is read back in UTC.
const toDateString = (timestamp: number) => new Date(timestamp).toISOString().slice(0, 10);

interface UpdateLegalHoldFormProps {
    updateLegalHold: (data: UpdateLegalHold) => Promise<unknown>;
    onClose: () => void;
    visible: boolean;
    legalHold: LegalHold | null;
}

const UpdateLegalHoldForm = (props: UpdateLegalHoldFormProps) => {
    const dispatch = useDispatch();
    const {users: storedUsers, groups: storedGroups, complete} = useLegalHoldEntities(props.legalHold);

    const [id, setId] = useState('');
    const [displayName, setDisplayName] = useState('');
    const [users, setUsers] = useState<UserProfile[]>([]);
    const [groups, setGroups] = useState<Group[]>([]);
    const [startsAt, setStartsAt] = useState('');
    const [endsAt, setEndsAt] = useState('');
    const [saving, setSaving] = useState(false);
    const [includePublicChannels, setIncludePublicChannels] = useState(false);
    const [serverError, setServerError] = useState('');
    const [loadError, setLoadError] = useState('');
    const [targetsRequested, setTargetsRequested] = useState(false);

    // The form shows the targets only after the store holds all of them. Otherwise a save removes the missing targets.
    const targetsReady = complete || targetsRequested;
    const unresolvedTargets = targetsReady ? ((props.legalHold?.user_ids?.length ?? 0) - storedUsers.length) +
        ((props.legalHold?.group_ids?.length ?? 0) - storedGroups.length) : 0;

    const resetForm = () => {
        setId('');
        setDisplayName('');
        setEndsAt('');
        setUsers([]);
        setGroups([]);
        setServerError('');
        setLoadError('');
        setTargetsRequested(false);
        setIncludePublicChannels(false);
        setSaving(false);
    };

    // Load the targeted users and groups that the store does not hold yet.
    useEffect(() => {
        const legalHold = props.legalHold;
        if (!props.visible || !legalHold || complete || targetsRequested) {
            return undefined;
        }

        let cancelled = false;
        type Result = {error?: unknown} | undefined;
        Promise.all([
            dispatch(getMissingProfilesByIds(legalHold.user_ids ?? [])) as unknown as Promise<Result>,
            dispatch(getMissingGroupsByIds(legalHold.group_ids ?? [])) as unknown as Promise<Result>,
        ]).then((results) => {
            if (cancelled) {
                return;
            }
            if (results.some((result) => result?.error)) {
                setLoadError('Could not load all the targeted users and groups. Close this dialog and try again.');
                return;
            }
            setTargetsRequested(true);
        }).catch(() => {
            if (!cancelled) {
                setLoadError('Could not load all the targeted users and groups. Close this dialog and try again.');
            }
        });

        return () => {
            cancelled = true;
        };
    }, [dispatch, props.legalHold, props.visible, complete, targetsRequested]);

    // Populate initial form field values when the legal hold being edited changes.
    useEffect(() => {
        if (!props.legalHold || props.legalHold.id === id || !targetsReady) {
            return;
        }

        setId(props.legalHold.id);
        setDisplayName(props.legalHold.display_name);
        setUsers(storedUsers);
        setGroups(storedGroups);
        setIncludePublicChannels(props.legalHold.include_public_channels);

        if (props.legalHold.starts_at) {
            setStartsAt(toDateString(props.legalHold.starts_at));
        }

        if (props.legalHold.ends_at) {
            setEndsAt(toDateString(props.legalHold.ends_at));
        }
    }, [props.legalHold, storedUsers, storedGroups, id, targetsReady]);

    const onSave = (e: FormEvent) => {
        e.preventDefault();
        if (saving || !props.legalHold || !canUpdate()) {
            return;
        }
        setSaving(true);

        const data = {
            id: props.legalHold.id,
            user_ids: users.map((user) => user.id),
            group_ids: groups.map((group) => group.id),
            ends_at: endsAt ? new Date(endsAt).getTime() : 0,
            include_public_channels: includePublicChannels,
            display_name: displayName,
        };

        props.updateLegalHold(data).then(() => {
            resetForm();
            props.onClose();
        }).catch((error) => {
            setSaving(false);
            setServerError(String(error));
        });
    };

    // A close request during a save is ignored, so the pending request cannot reset a form that was reopened.
    const onCancel = () => {
        if (saving) {
            return;
        }
        resetForm();
        props.onClose();
    };

    const canUpdate = () => {
        if (!targetsReady) {
            return false;
        }
        if (endsAt !== '' && startsAt >= endsAt) {
            return false;
        }
        if (displayName.length < 2 || displayName.length > 64) {
            return false;
        }

        if (users.length < 1 && groups.length < 1) {
            return false;
        }

        return true;
    };

    const footer = (
        <>
            <Button
                type='button'
                emphasis='tertiary'
                disabled={saving}
                onClick={onCancel}
            >
                {'Cancel'}
            </Button>
            <Button
                type='submit'
                form={FORM_ID}
                loading={saving}
                disabled={!canUpdate()}
            >
                {'Update legal hold'}
            </Button>
        </>
    );

    return (
        <ModalOverlay
            open={props.visible && Boolean(props.legalHold)}
            onClose={onCancel}
        >
            <Modal
                open={props.visible && Boolean(props.legalHold)}
                onClose={onCancel}
                title='Update legal hold'
                size='medium'
                footer={footer}
            >
                <form
                    id={FORM_ID}
                    className='legal-hold-form'
                    onSubmit={onSave}
                >
                    {loadError && (
                        <SectionNotice
                            role='alert'
                            type='danger'
                            title={loadError}
                        />
                    )}
                    {unresolvedTargets > 0 && (
                        <SectionNotice
                            type='warning'
                            title={`${unresolvedTargets} targeted users or groups no longer exist. Updating the legal hold removes them.`}
                        />
                    )}
                    {serverError && (
                        <SectionNotice
                            role='alert'
                            type='danger'
                            title={serverError}
                        />
                    )}
                    <LegalHoldFormFields
                        displayName={displayName}
                        onDisplayNameChange={setDisplayName}
                        users={users}
                        onUsersChange={setUsers}
                        groups={groups}
                        onGroupsChange={setGroups}
                        includePublicChannels={includePublicChannels}
                        onIncludePublicChannelsChange={setIncludePublicChannels}
                        publicChannelsNotice='It is possible for users to access public content without becoming members of a public channel. This setting only captures public channels the users are members of.'
                        startsAt={startsAt}
                        startsAtLabel='Starting at'
                        endsAt={endsAt}
                        onEndsAtChange={setEndsAt}
                    />
                </form>
            </Modal>
        </ModalOverlay>
    );
};

export default UpdateLegalHoldForm;
