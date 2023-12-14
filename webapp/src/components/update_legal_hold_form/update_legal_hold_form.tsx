import React, {useEffect, useState} from 'react';
import dayjs from 'dayjs';

import {UserProfile} from 'mattermost-redux/types/users';

import UsersInput from '@/components/users_input';
import {LegalHold, UpdateLegalHold} from '@/types';
import {GenericModal} from '@/components/mattermost-webapp/generic_modal/generic_modal';
import Input from '@/components/mattermost-webapp/input/input';

import '../create_legal_hold_form.scss';

interface UpdateLegalHoldFormProps {
    updateLegalHold: (data: UpdateLegalHold) => Promise<any>;
    onExited: () => void;
    visible: boolean;
    legalHold: LegalHold|null;
    users: Array<UserProfile>;
}

const UpdateLegalHoldForm = (props: UpdateLegalHoldFormProps) => {
    const [id, setId] = useState("");
    const [displayName, setDisplayName] = useState('');
    const [users, setUsers] = useState(Array<UserProfile>());
    const [startsAt, setStartsAt] = useState('');
    const [endsAt, setEndsAt] = useState('');
    const [saving, setSaving] = useState(false);
    const [serverError, setServerError] = useState('');

    const displayNameChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        setDisplayName(e.target.value);
    };

    const endsAtChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        setEndsAt(e.target.value);
    };

    const resetForm = () => {
        setDisplayName('');
        setEndsAt('');
        setUsers([]);
        setSaving(false);
        setServerError('');
    };

    // Populate initial form field values when the Legal Hold being edited changes.
    useEffect(() => {
        if (props.legalHold) {
            if (props.legalHold.id == id) {
                return
            }

            setId(props.legalHold.id);
            setDisplayName(props.legalHold?.display_name);
            setUsers(props.users);

            if (props.legalHold.starts_at) {
                const startsAt = dayjs(props.legalHold.starts_at).format('YYYY-MM-DD');
                setStartsAt(startsAt);
            }

            if (props.legalHold.ends_at) {
                const endsAt = dayjs(props.legalHold.ends_at).format('YYYY-MM-DD');
                setEndsAt(endsAt);
            }
        }
    }, [props.legalHold, props.users]);

    const onSave = () => {
        if (saving) {
            return;
        }
        setSaving(true);

        if (!props.legalHold) {
            return
        }

        const data = {
            id: props.legalHold.id,
            user_ids: users.map((user) => user.id),
            ends_at: (new Date(endsAt)).getTime(),
            display_name: displayName,
        };

        props.updateLegalHold(data).then((response) => {
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

        if (users.length < 1) {
            return false;
        }

        return true;
    };

    if (!props.legalHold) {
        return <div></div>;
    }

    return (
        <GenericModal
            id='edit-legal-hold-modal'
            className='edit-legal-hold-modal'
            modalHeaderText='Update new legal hold'
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
                        <UsersInput
                            placeholder='@username1 @username2'
                            users={users}
                            onChange={setUsers}
                        />
                    </div>
                    <div
                        style={{
                            display: 'flex',
                            columnGap: '20px',
                        }}
                    >
                        <div>
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
                        </div>
                        <Input
                            type='date'
                            autoComplete='off'
                            autoFocus={false}
                            required={true}
                            name={'Ending at'}
                            label={'Ending at'}
                            placeholder={'Ending at'}
                            limit={64}
                            value={endsAt}
                            onChange={endsAtChanged}
                            onBlur={endsAtChanged}
                            containerClassName={'create-legal-hold-container'}
                            inputClassName={'create-legal-hold-input'}
                        />
                    </div>
                </div>
            </div>
        </GenericModal>
    );
};

export default UpdateLegalHoldForm;

