import React from "react";
import {useState} from "react";
import UsersInput from "@/components/users_input";
import {IntlProvider} from "react-intl";
import Client from "@/client";

import SaveButton from "@/components/mattermost-webapp/save_button"

const CreateLegalHoldForm = () => {
    const [displayName, setDisplayName] = useState("");
    const [users, setUsers] = useState(Array<string>());
    const [startsAt, setStartsAt] = useState(0);
    const [endsAt, setEndsAt] = useState(0);
    const [saving, setSaving] = useState(false);

    const displayNameChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        setDisplayName(e.target.value);
    };

    const startsAtChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        console.log(e.target.value);
        setStartsAt((new Date(e.target.value)).getTime());
        console.log((new Date(e.target.value)).getTime());
    };

    const endsAtChanged = (e: React.ChangeEvent<HTMLInputElement>) => {
        console.log(e.target.value);
        setEndsAt((new Date(e.target.value)).getTime());
        console.log((new Date(e.target.value)).getTime());
    };

    const saveClicked = () => {
        console.log("Save Clicked");
        setSaving(true);
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

export default CreateLegalHoldForm;

