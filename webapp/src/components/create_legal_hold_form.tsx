import {useState} from 'react';
import type {FormEvent} from 'react';

import {Button} from '@mattermost/compass-ui/components/button';
import {Modal, ModalOverlay} from '@mattermost/compass-ui/components/modal';
import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';
import type {Group} from '@mattermost/types/groups';
import type {UserProfile} from '@mattermost/types/users';

import LegalHoldFormFields from '@/components/legal_hold_form_fields';
import type {CreateLegalHold} from '@/types';

import './legal_hold_form.css';

const FORM_ID = 'new-legal-hold-form';

interface CreateLegalHoldFormProps {
    createLegalHold: (data: CreateLegalHold) => Promise<unknown>;
    onClose: () => void;
    visible: boolean;
}

const slugify = (data: string) => {
    return data.
        replace(/[^0-9a-zA-Z _-]/g, '').
        replace(/[ _]/g, '-').
        toLowerCase();
};

const CreateLegalHoldForm = (props: CreateLegalHoldFormProps) => {
    const [displayName, setDisplayName] = useState('');
    const [users, setUsers] = useState<UserProfile[]>([]);
    const [groups, setGroups] = useState<Group[]>([]);
    const [startsAt, setStartsAt] = useState('');
    const [endsAt, setEndsAt] = useState('');
    const [saving, setSaving] = useState(false);
    const [includePublicChannels, setIncludePublicChannels] = useState(false);
    const [serverError, setServerError] = useState('');

    const resetForm = () => {
        setDisplayName('');
        setStartsAt('');
        setEndsAt('');
        setUsers([]);
        setGroups([]);
        setSaving(false);
        setIncludePublicChannels(false);
        setServerError('');
    };

    const onSave = (e: FormEvent) => {
        e.preventDefault();
        if (saving || !canCreate()) {
            return;
        }
        setSaving(true);

        const data: CreateLegalHold = {
            user_ids: users.map((user) => user.id),
            group_ids: groups.map((group) => group.id),
            ends_at: endsAt ? new Date(endsAt).getTime() : 0,
            starts_at: new Date(startsAt).getTime(),
            display_name: displayName,
            include_public_channels: includePublicChannels,
            name: slugify(displayName),
        };

        props.createLegalHold(data).then(() => {
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

    const canCreate = () => {
        if (startsAt === '') {
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
                disabled={!canCreate()}
            >
                {'Create legal hold'}
            </Button>
        </>
    );

    return (
        <ModalOverlay
            open={props.visible}
            onClose={onCancel}
        >
            <Modal
                open={props.visible}
                onClose={onCancel}
                title='Create a new legal hold'
                size='medium'
                footer={footer}
            >
                <form
                    id={FORM_ID}
                    className='legal-hold-form'
                    onSubmit={onSave}
                >
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
                        publicChannelsNotice='It is possible for users to access public content without becoming members of a public channel. This setting only captures public channels that users are members of.'
                        startsAt={startsAt}
                        startsAtLabel='Starting from'
                        onStartsAtChange={setStartsAt}
                        endsAt={endsAt}
                        onEndsAtChange={setEndsAt}
                    />
                </form>
            </Modal>
        </ModalOverlay>
    );
};

export default CreateLegalHoldForm;
