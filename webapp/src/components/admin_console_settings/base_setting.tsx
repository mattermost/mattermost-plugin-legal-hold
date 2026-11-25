import type {PropsWithChildren} from 'react';

import {FieldMessage} from '@mattermost/compass-ui/components/field-message';

import './admin_console_settings.css';

type Props = PropsWithChildren<{
    id: string;
    name: string;
    helpText: string;

    // Set when the field is a group of radios, which a label cannot point at with htmlFor.
    group?: boolean;
}>;

// One admin console setting row: the label sits in the left column and the field in the right one.
const BaseSetting = (props: Props) => {
    const labelId = `${props.id}-label`;

    return (
        <div
            id={`legal-hold-admin-console-setting-${props.id}`}
            className={'legal-hold-setting' + (props.group ? ' legal-hold-setting--radios' : '')}
        >
            {!props.name && <span className='legal-hold-setting__label'/>}
            {props.name && props.group && (
                <span
                    id={labelId}
                    data-testid={labelId}
                    className='legal-hold-setting__label'
                >
                    {props.name}
                </span>
            )}
            {props.name && !props.group && (
                <label
                    id={labelId}
                    data-testid={labelId}
                    htmlFor={props.id}
                    className='legal-hold-setting__label'
                >
                    {props.name}
                </label>
            )}
            <div
                className='legal-hold-setting__field'
                role={props.group ? 'radiogroup' : undefined}
                aria-labelledby={props.group ? labelId : undefined}
            >
                {props.children}
                {props.helpText && (
                    <FieldMessage
                        tone='info'
                        message={props.helpText}
                    />
                )}
            </div>
        </div>
    );
};

export default BaseSetting;
