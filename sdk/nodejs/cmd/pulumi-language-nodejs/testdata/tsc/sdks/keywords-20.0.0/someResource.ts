// *** WARNING: this file was generated by pulumi-language-nodejs. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

import * as pulumi from "@pulumi/pulumi";
import * as utilities from "./utilities";

export class SomeResource extends pulumi.CustomResource {
    /**
     * Get an existing SomeResource resource's state with the given name, ID, and optional extra
     * properties used to qualify the lookup.
     *
     * @param name The _unique_ name of the resulting resource.
     * @param id The _unique_ provider ID of the resource to lookup.
     * @param opts Optional settings to control the behavior of the CustomResource.
     */
    public static get(name: string, id: pulumi.Input<pulumi.ID>, opts?: pulumi.CustomResourceOptions): SomeResource {
        return new SomeResource(name, undefined as any, { ...opts, id: id });
    }

    /** @internal */
    public static readonly __pulumiType = 'keywords:index:SomeResource';

    /**
     * Returns true if the given object is an instance of SomeResource.  This is designed to work even
     * when multiple copies of the Pulumi SDK have been loaded into the same process.
     */
    public static isInstance(obj: any): obj is SomeResource {
        if (obj === undefined || obj === null) {
            return false;
        }
        return obj['__pulumiType'] === SomeResource.__pulumiType;
    }

    public readonly builtins!: pulumi.Output<string>;
    public readonly property!: pulumi.Output<string>;

    /**
     * Create a SomeResource resource with the given unique name, arguments, and options.
     *
     * @param name The _unique_ name of the resource.
     * @param args The arguments to use to populate this resource's properties.
     * @param opts A bag of options that control this resource's behavior.
     */
    constructor(name: string, args: SomeResourceArgs, opts?: pulumi.CustomResourceOptions) {
        let resourceInputs: pulumi.Inputs = {};
        opts = opts || {};
        if (!opts.id) {
            if ((!args || args.builtins === undefined) && !opts.urn) {
                throw new Error("Missing required property 'builtins'");
            }
            if ((!args || args.property === undefined) && !opts.urn) {
                throw new Error("Missing required property 'property'");
            }
            resourceInputs["builtins"] = args ? args.builtins : undefined;
            resourceInputs["property"] = args ? args.property : undefined;
        } else {
            resourceInputs["builtins"] = undefined /*out*/;
            resourceInputs["property"] = undefined /*out*/;
        }
        opts = pulumi.mergeOptions(utilities.resourceOptsDefaults(), opts);
        super(SomeResource.__pulumiType, name, resourceInputs, opts);
    }
}

/**
 * The set of arguments for constructing a SomeResource resource.
 */
export interface SomeResourceArgs {
    builtins: pulumi.Input<string>;
    property: pulumi.Input<string>;
}
