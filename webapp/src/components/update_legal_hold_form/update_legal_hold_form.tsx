import dayjs from 'dayjs';
import React, {useEffect, useState} from 'react';

import {UserProfile} from 'mattermost-redux/types/users';
import {Group} from 'mattermost-redux/types/groups';

import {GenericModal} from '@/components/mattermost-webapp/generic_modal/generic_modal';
import Input from '@/components/mattermost-webapp/input/input';
import UsersInput from '@/components/users_input';
import GroupsInput from '@/components/groups_input';
import {LegalHold, UpdateLegalHold} from '@/types';
import {isValidDate} from '@/utils/date_validation';

import '../create_legal_hold_form.scss';

interface UpdateLegalHoldFormProps {
    updateLegalHold: (data: UpdateLegalHold) => Promise<any>;
    onExited: () => void;
    visible: boolean;
    legalHold: LegalHold | null;
    users: Array<UserProfile>;
    groups: Array<Group>;
}

const UpdateLegalHoldForm = (props: UpdateLegalHoldFormProps) => {
    const [id, setId] = useState('');
    const [displayName, setDisplayName] = useState('');
    const [users, setUsers] = useState(Array<UserProfile>());
    const [groups, setGroups] = useState(Array<Group>());
    const [startsAt, setStartsAt] = useState('');
    const [endsAt, setEndsAt] = useState('');
    const [saving, setSaving] = useState(false);
    const [includePublicChannels, setIncludePublicChannels] = useState(false);
    const [serverError, setServerError] = useState('');
    const [endsAtInvalid, setEndsAtInvalid] = useState(false);

    const displayNameChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        setDisplayName(e.target.value);
    };

    const endsAtChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        const dateValue = e.target.value;
        if (dateValue && !isValidDate(dateValue)) {
            setEndsAtInvalid(true);
            return;
        }
        setEndsAtInvalid(false);
        setEndsAt(dateValue);
    };

    const includePublicChannelsChanged: (e: React.ChangeEvent<HTMLInputElement>) => void = (e) => {
        setIncludePublicChannels(e.target.checked);
    };

    const resetForm = () => {
        setId('');
        setDisplayName('');
        setEndsAt('');
        setUsers([]);
        setGroups([]);
        setServerError('');
        setIncludePublicChannels(false);
        setSaving(false);
        setEndsAtInvalid(false);
    };

    // Populate initial form field values when the Legal Hold being edited changes.
    useEffect(() => {
        if (props.legalHold) {
            if (props.legalHold.id === id) {
                return;
            }

            setId(props.legalHold.id);
            setDisplayName(props.legalHold?.display_name);
            setUsers(props.users);
            setGroups(props.groups);
            setIncludePublicChannels(props.legalHold.include_public_channels);

            if (props.legalHold.starts_at) {
                const startsAtString = dayjs(props.legalHold.starts_at).format('YYYY-MM-DD');
                setStartsAt(startsAtString);
            }

            if (props.legalHold.ends_at) {
                const endsAtString = dayjs(props.legalHold.ends_at).format('YYYY-MM-DD');
                setEndsAt(endsAtString);
            }
        }
    }, [props.legalHold, props.users, props.groups, props.visible, id]);

    const onSave = () => {
        if (saving) {
            return;
        }
        setSaving(true);

        if (!props.legalHold) {
            return;
        }

        const data = {
            id: props.legalHold.id,
            user_ids: users.map((user) => user.id),
            group_ids: groups.map((group) => group.id),
            ends_at: (new Date(endsAt)).getTime(),
            include_public_channels: includePublicChannels,
            display_name: displayName,
        };

        props.updateLegalHold(data).then(() => {
            resetForm();
            props.onExited();
        }).catch((error) => {
            setSaving(false);
            setServerError(error.toString());
        });
    };

    const onCancel = () => {
        resetForm();
        props.onExited();
    };

    const canUpdate = () => {
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

    if (!props.legalHold) {
        return <div/>;
    }

    return (
        <GenericModal
            id='edit-legal-hold-modal'
            className='edit-legal-hold-modal'
            modalHeaderText='Update legal hold'
            confirmButtonText='Update legal hold'
            cancelButtonText='Cancel'
            errorText={serverError}
            isConfirmDisabled={!canUpdate()}
            autoCloseOnConfirmButton={false}
            compassDesign={true}
            handleConfirm={onSave}
            handleEnterKeyPress={onSave}
            handleCancel={onCancel}
            onExited={onCancel}
            show={props.visible}
        >
            <div>
                <div
                    style={{
                        display: 'flex',
                        flexDirection: 'column',
                        rowGap: '20px',
                    }}
                >
                    <Input
                        type='text'
                        autoComplete='off'
                        autoFocus={false}
                        required={true}
                        name={'Name'}
                        label={'Name'}
                        placeholder={'New Legal Hold...'}
                        limit={64}
                        value={displayName}
                        onChange={displayNameChanged}
                        onBlur={displayNameChanged}
                        containerClassName={'create-legal-hold-container'}
                        inputClassName={'create-legal-hold-input'}
                    />
                    <div>
                        <label>{'Users'}</label>
                        <UsersInput
                            placeholder='@username1 @username2'
                            users={users}
                            onChange={setUsers}
                        />
                    </div>
                    <div>
                        <label>{'LDAP Groups'}</label>
                        <GroupsInput
                            placeholder='group1 group2'
                            groups={groups}
                            onChange={setGroups}
                        />
                    </div>
                    <div
                        style={{
                            display: 'flex',
                            columnGap: '20px',
                        }}
                    >
                        <input
                            type='checkbox'
                            id='legal-hold-include-public-channels'
                            checked={includePublicChannels}
                            onChange={includePublicChannelsChanged}
                            className={'create-legal-hold-checkbox'}
                        />
                        <label htmlFor={'legal-hold-include-public-channels'}>
                            {'Include public channels'}
                        </label>
                    </div>
                    <div
                        style={{
                            display: (includePublicChannels) ? 'block' : 'none',
                            marginTop: '-20px',
                            marginBottom: '20px',
                        }}
                    >
                        <i className='icon icon-alert-outline'/>
                        <span>{'It is possible for users to access public content without becoming members of a public channel. This setting only captures public channels the users are members of.'}</span>
                    </div>
                    <div
                        style={{
                            display: 'flex',
                            columnGap: '20px',
                        }}
                    >
                        <Input
                            type='date'
                            autoComplete='off'
                            autoFocus={false}
                            required={true}
                            name={'Starting at'}
                            label={'Starting at'}
                            placeholder={'Starting at'}
                            limit={64}
                            value={startsAt}
                            containerClassName={'create-legal-hold-container'}
                            inputClassName={'create-legal-hold-input'}
                            disabled={true}
                        />
                        <Input
                            type='date'
                            autoComplete='off'
                            autoFocus={false}
                            required={false}
                            name={'Ending at'}
                            label={'Ending at'}
                            placeholder={'Ending at'}
                            limit={64}
                            value={endsAt}
                            onChange={endsAtChanged}
                            onBlur={endsAtChanged}
                            containerClassName={'create-legal-hold-container'}
                            inputClassName={'create-legal-hold-input'}
                            hasError={endsAtInvalid}
                        />
                    </div>
                </div>
            </div>
        </GenericModal>
    );
};

export default UpdateLegalHoldForm;
