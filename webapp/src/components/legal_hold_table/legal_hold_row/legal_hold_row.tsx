import {LegalHold} from "@/types";
import React from "react";
import {UserProfile} from "mattermost-redux/types/users";


interface LegalHoldRowProps {
    legalHold: LegalHold;
    users: UserProfile[]
}

const LegalHoldRow = (props: LegalHoldRowProps) => {
    const lh = props.legalHold;
    const startsAt = (new Date(lh.starts_at)).toLocaleDateString();
    const endsAt = (new Date(lh.ends_at)).toLocaleDateString();

    return (
        <React.Fragment>
            <div>{lh.name}</div>
            <div>{lh.display_name}</div>
            <div>{startsAt}</div>
            <div>{endsAt}</div>
            <div>{props.users.map((user) => `@${user.username} `)}</div>
            <div><a href="#">Edit</a></div>
        </React.Fragment>
    );
}

export default LegalHoldRow;
