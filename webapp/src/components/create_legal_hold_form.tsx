import React from "react";
import {useState} from "react";
import UsersInput from "@/components/users_input";
import {UserProfile} from 'mattermost-redux/types/users';
import {Modal} from 'react-bootstrap';

import SaveButton from "@/components/mattermost-webapp/save_button"
import {CreateLegalHold} from "@/types";
import {GenericModal} from "@/components/mattermost-webapp/generic_modal/generic_modal";

interface CreateLegalHoldFormProps {
    createLegalHold: (data: CreateLegalHold) => Promise<any>;
    onExited: () => void;
    visible: boolean;
}

const CreateLegalHoldForm = (props: CreateLegalHoldFormProps) => {
    const [displayName, setDisplayName] = useState("");
    const [users, setUsers] = useState(Array<UserProfile>());
    const [startsAt, setStartsAt] = useState("");
    const [endsAt, setEndsAt] = useState("");
    const [saving, setSaving] = useState(false);
    const [serverError, setServerError] = useState("");

    const displayNameChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        setDisplayName(e.target.value);
    };

    const startsAtChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        setStartsAt(e.target.value);
    };

    const endsAtChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        setEndsAt(e.target.value);
    };

    const saveClicked = () => {
        if (saving) return;
        setSaving(true);

        const data = {
            user_ids: users.map((user) => user.id),
            ends_at: (new Date(endsAt)).getTime(),
            starts_at: (new Date(startsAt)).getTime(),
            display_name: displayName,
            name: slugify(displayName),
        };

        props.createLegalHold(data).then(response => {
            setDisplayName("");
            setStartsAt("");
            setEndsAt("");
            setUsers([]);
            setSaving(false);
            props.onExited();
        }).catch(error => {
            setSaving(false);
            setServerError(error.toString());
        });
    };

    // TODO: Implement validation.
    const canCreate = true;

    return (
        <GenericModal
            id='new-legal-hold-modal'
            className='new-legal-hold-modal'
            modalHeaderText="Create a new legal hold"
            confirmButtonText="Create legal hold"
            cancelButtonText="Cancel"
            errorText={serverError}
            isConfirmDisabled={!canCreate}
            autoCloseOnConfirmButton={false}
            compassDesign={true}
            handleConfirm={saveClicked}
            handleEnterKeyPress={saveClicked}
            handleCancel={props.onExited}
            onExited={props.onExited}
            show={props.visible}
        >
            <div>
                <div style={{
                    display: "grid",
                    gridTemplateColumns: "20% auto",
                }}>
                    <div>
                        Display Name:
                    </div>
                    <div>
                        <input
                            type={"text"}
                            className="form-control"
                            onChange={displayNameChanged}
                            value={displayName}
                        />
                    </div>
                    <div>
                        Users:
                    </div>
                    <div>
                        <UsersInput
                            placeholder='@username1 @username2'
                            users={users}
                            onChange={setUsers}
                        />
                    </div>
                    <div>
                        Starting From:
                    </div>
                    <div>
                        <input
                            type={"date"}
                            onChange={startsAtChanged}
                            className="form-control"
                            value={startsAt}
                        />
                    </div>
                    <div>
                        Ending At:
                    </div>
                    <div>
                        <input
                            type={"date"}
                            onChange={endsAtChanged}
                            className="form-control"
                            value={endsAt}
                        />
                    </div>
                </div>
            </div>
        </GenericModal>
    );
};

const slugify = (data: string) => {
    return data
        .replace(/[^0-9a-zA-Z _-]/g, "")
        .replace(/[ _]/g, "-")
        .toLowerCase();
}

export default CreateLegalHoldForm;

