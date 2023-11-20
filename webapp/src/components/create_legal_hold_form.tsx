import React from "react";
import {useState} from "react";
import UsersInput from "@/components/users_input";
import {IntlProvider} from "react-intl";
import {UserProfile} from 'mattermost-redux/types/users';

import SaveButton from "@/components/mattermost-webapp/save_button"
import {CreateLegalHold} from "@/types";

interface CreateLegalHoldFormProps {
    createLegalHold: (data: CreateLegalHold) => Promise<any>;
}

const CreateLegalHoldForm = (props: CreateLegalHoldFormProps) => {
    const [displayName, setDisplayName] = useState("");
    const [users, setUsers] = useState(Array<UserProfile>());
    const [startsAt, setStartsAt] = useState("");
    const [endsAt, setEndsAt] = useState("");
    const [saving, setSaving] = useState(false);

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

        props.createLegalHold(data).
            then(response => {
                setDisplayName("");
                setStartsAt("");
                setEndsAt("");
                setUsers([]);
                setSaving(false);
        }).catch(error => {
            setSaving(false);
        });
    };

    return (
        <IntlProvider locale="en-US">
            <div>
                <h3>Create a New Legal Hold</h3>
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
                    <div/>
                    <div>
                        <SaveButton
                            onClick={saveClicked}
                            saving={saving}
                            disabled={false}
                            savingMessage={"Creating Legal Hold..."}
                        />
                    </div>
                </div>
            </div>
        </IntlProvider>
    );
};

const slugify = (data: string) => {
    return data
        .replace(/[^0-9a-zA-Z _-]/g, "")
        .replace(/[ _]/g, "-")
        .toLowerCase();
}

export default CreateLegalHoldForm;

